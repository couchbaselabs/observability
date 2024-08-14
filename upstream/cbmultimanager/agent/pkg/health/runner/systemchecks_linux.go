// Copyright (C) 2022 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package runner

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/couchbase/tools-common/cbrest"
	"github.com/couchbase/tools-common/cbvalue"
	"go.uber.org/zap"
	"golang.org/x/exp/slices"
	"golang.org/x/sys/unix"

	"github.com/couchbaselabs/cbmultimanager/agent/pkg/bootstrap"
	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/values"
)

var thpLocations = [4]string{
	"/sys/kernel/mm/transparent_hugepage/enabled",
	"/sys/kernel/mm/transparent_hugepage/defrag",
	"/sys/kernel/mm/redhat_transparent_hugepage/enabled",
	"/sys/kernel/mm/redhat_transparent_hugepage/defrag",
}

// getSystemCheckers returns all the checker functions to be run by the agent
func getSystemCheckers() map[string]checkerFn {
	result := map[string]checkerFn{
		values.CheckTHP:               checkTHP,
		values.CheckProcessLimits:     checkProcessLimits,
		values.CheckSupportedOS:       checkOSRelease,
		values.CheckPortStatus:        checkPortStatus,
		values.CheckAutoFailoverForVM: checkAutoFailoverForVM,
	}
	for key, val := range getUniversalCheckers() {
		result[key] = val
	}
	return result
}

func checkTHP(node *bootstrap.Node) *values.WrappedCheckerResult {
	result := &values.WrappedCheckerResult{
		Result: &values.CheckerResult{
			Name:   values.CheckTHP,
			Status: values.MissingCheckerStatus,
			Time:   time.Now().UTC(),
		},
		Cluster: node.RestClient().ClusterUUID(),
		Node:    node.UUID(),
	}

	for _, location := range thpLocations {
		out, err := os.ReadFile(location)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}

			result.Error = fmt.Errorf("could not read THP settings at '%s': '%w'", location, err)
			return result
		}

		res := strings.TrimSpace(string(out))

		// the file format is something along the lines of
		// [always] madvise never
		// where the selected value is the one in between brackets
		start, end := strings.Index(res, "["), strings.Index(res, "]")
		if start == -1 || end == -1 {
			// invalid format so ignore
			continue
		}

		out, _ = json.Marshal(res)
		result.Result.Value = out

		switch res[start+1 : end] {
		case "madvise", "never":
			result.Result.Status = values.GoodCheckerStatus
		case "always":
			result.Result.Status = values.InfoCheckerStatus
			result.Result.Remediation = "THP is enabled. Set THP to madvise or never."
		default:
			result.Error = fmt.Errorf("did not recognize THP setting '%s'", res[start+1:end])
		}

		return result
	}

	result.Error = fmt.Errorf("could not find THP settings")
	return result
}

const (
	recommendedMaxOpenFiles = 40_960
	recommendedMaxProcesses = 10_000
)

func checkProcessLimits(node *bootstrap.Node) *values.WrappedCheckerResult {
	result := &values.WrappedCheckerResult{
		Result: &values.CheckerResult{
			Name:   values.CheckProcessLimits,
			Status: values.MissingCheckerStatus,
			Time:   time.Now().UTC(),
		},
		Cluster: node.RestClient().ClusterUUID(),
		Node:    node.UUID(),
	}

	// Easiest way to find the process we're looking for is to walk /proc/<numbers> until we find one with
	// a `cmdline` containing `ns_babysitter`.
	procFile, err := os.Open("/proc")
	if err != nil {
		result.Error = fmt.Errorf("could not open /proc: %w", err)
		return result
	}
	defer procFile.Close()
	dirs, err := procFile.Readdirnames(0)
	if err != nil {
		result.Error = fmt.Errorf("could not list processes: %w", err)
		return result
	}
	var babysitterPid int
	for _, dir := range dirs {
		pid, err := strconv.ParseInt(dir, 10, 0)
		if err != nil {
			continue // not a valid PID
		}
		// Check its cmdline - if we try to look at non-couchbase processes we'll get an EPERM
		cmdlineFile, err := os.Open(fmt.Sprintf("/proc/%d/cmdline", pid))
		if err != nil {
			continue
		}
		cmdline, err := io.ReadAll(cmdlineFile)
		if err != nil {
			cmdlineFile.Close()
			continue
		}
		cmdlineFile.Close()
		if strings.Contains(string(cmdline), "\x00-ns_babysitter\x00") {
			babysitterPid = int(pid)
			break
		}
	}
	if babysitterPid == 0 {
		result.Error = fmt.Errorf("could not find babysitter process")
		return result
	}

	var fileLimits unix.Rlimit
	if err := unix.Prlimit(babysitterPid, unix.RLIMIT_NOFILE, nil, &fileLimits); err != nil {
		result.Error = fmt.Errorf("could not determine open file limits: %w", err)
	}

	if fileLimits.Cur < recommendedMaxOpenFiles {
		result.Result.Status = values.WarnCheckerStatus
		result.Result.Remediation = "Increase the maximum open files limit for Couchbase Server processes."
	} else {
		result.Result.Status = values.GoodCheckerStatus
	}

	var procLimits unix.Rlimit
	if err := unix.Prlimit(babysitterPid, unix.RLIMIT_NPROC, nil, &procLimits); err != nil {
		result.Error = fmt.Errorf("could not determine process limits: %w", err)
	}

	if procLimits.Cur < recommendedMaxProcesses {
		result.Result.Status = values.WarnCheckerStatus
		if result.Result.Remediation != "" {
			result.Result.Remediation += " "
		}
		result.Result.Remediation += "Increase the user processes limit for Couchbase Server processes."
	}

	result.Result.Value, _ = json.Marshal(struct {
		OpenFiles unix.Rlimit
		Processes unix.Rlimit
	}{
		OpenFiles: fileLimits,
		Processes: procLimits,
	})
	return result
}

var _osReleaseFns = map[string]func() (string, error){
	"lsb_release": func() (string, error) {
		lsbReleaseExe, err := exec.LookPath("lsb_release")
		if err != nil {
			return "", err
		}
		cmd := exec.Command(lsbReleaseExe, "-ds")
		result, err := cmd.Output()
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(result)), nil
	},
	"redhat-release": func() (string, error) {
		contents, err := os.ReadFile("/etc/redhat-release")
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(contents)), nil
	},
	"SuSE-release": func() (string, error) {
		contents, err := os.ReadFile("/etc/SuSE-release")
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(contents)), nil
	},
	"system-release": func() (string, error) {
		contents, err := os.ReadFile("/etc/system-release")
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(contents)), nil
	},
}

func findOSRelease() (string, error) {
	for name, fn := range _osReleaseFns {
		result, err := fn()
		if err != nil {
			zap.S().Named("System Checks Linux").
				Debugw("Failed to execute OS release function", "fn", name, "error", err)
			continue
		}
		return result, nil
	}
	return "", fmt.Errorf("couldn't determine OS release")
}

func checkOSRelease(node *bootstrap.Node) *values.WrappedCheckerResult {
	result := &values.WrappedCheckerResult{
		Result: &values.CheckerResult{
			Name:   values.CheckSupportedOS,
			Status: values.MissingCheckerStatus,
			Time:   time.Now().UTC(),
		},
		Cluster: node.RestClient().ClusterUUID(),
		Node:    node.UUID(),
	}

	osRel, err := findOSRelease()
	if err != nil {
		result.Error = err
		return result
	}
	result.Result.Value, _ = json.Marshal(&osRel)

	// the REST client is created with ThisNodeOnly, so this will be the current node's version no matter what
	nodeVers := node.Version()
	var versData *values.Version
	for _, version := range values.GAVersions {
		// we don't care about the build number
		parts := strings.SplitN(version.Build, "-", 2)
		if nodeVers.Equal(cbvalue.Version(parts[0])) {
			versData = &version
			break
		}
	}
	if versData == nil {
		result.Result.Status = values.InfoCheckerStatus
		result.Result.Remediation = "Could not check if this OS is supported because this node is running an unknown " +
			"version of Couchbase Server. If the current version is a generally available build, " +
			"please update cbhealthagent."
		return result
	}
	for _, versionOS := range versData.OS {
		if strings.HasPrefix(osRel, versionOS.Prefix) {
			if versionOS.Deprecated {
				result.Result.Status = values.InfoCheckerStatus
				result.Result.Remediation = fmt.Sprintf("'%s' is deprecated for Couchbase Server %s. "+
					"Please upgrade to a supported OS.", osRel, nodeVers)
			} else {
				result.Result.Status = values.GoodCheckerStatus
			}
			return result
		}
	}
	result.Result.Status = values.InfoCheckerStatus
	result.Result.Remediation = fmt.Sprintf("'%s' is not a supported OS for Couchbase Server %s. "+
		"Please upgrade to a supported OS.", osRel, nodeVers)
	return result
}

// helper function for checking if an int exists in an array of ints
func containsInt(matches []int64, toMatch int64) bool {
	for _, match := range matches {
		if match == toMatch {
			return true
		}
	}

	return false
}

// function to find name of process using the pid
func findProcessName(pid int) string {
	// find process name using ps -p command
	cmd := exec.Command("ps", "-p", strconv.Itoa(pid))
	stdout, err := cmd.Output()
	if err != nil {
		zap.S().Warnw("(Port Status Check) Could not find the name for process",
			"pid", pid, "error", err)
		return ""
	}
	stdString := strings.Fields(string(stdout))
	return stdString[len(stdString)-1]
}

const listenState string = "0A"

// function to get all port and inode numbers in listen state from /proc/net/tcp
func getAllActivePortsAndInodeNumbers(netFile []byte, ports []int64) ([]int64, []string) {
	portNumbers := make([]int64, 0)
	inodeNumbers := make([]string, 0)

	netFileLines := strings.Split(string(netFile), "\n")
	for _, line := range netFileLines[1:] {
		lineSplit := strings.Fields(line)
		// only for ports in LISTEN state
		if len(lineSplit) > 9 && lineSplit[3] == listenState {
			portNumberHex := strings.Split(lineSplit[1], ":")
			if len(portNumberHex) <= 1 {
				zap.S().Warnw("(Port Status Check) Garbage value found in /proc/net/tcp. "+
					"Unable to parse port number from string", "string", lineSplit[1])
				continue // port number not parsable
			}
			portNumberDec, err := strconv.ParseInt(portNumberHex[1], 16, 32)
			if err != nil {
				zap.S().Warnw("(Port Status Check) Could not parse port number",
					"string", portNumberHex[1], "error", err)
				continue
			}
			if containsInt(ports, portNumberDec) {
				portNumbers = append(portNumbers, portNumberDec)
				inodeNumbers = append(inodeNumbers, lineSplit[9])
			}
		}
	}
	return portNumbers, inodeNumbers
}

// function to retrieve all service ports with/out SSL
func getAllPorts(node *bootstrap.Node) []int64 {
	ports := make([]int64, len(allServices)*2)
	for _, service := range allServices {
		servicePortWithoutSSL := node.Services().GetPort(service, false)
		servicePortWithSSL := node.Services().GetPort(service, true)
		if servicePortWithoutSSL != 0 {
			ports = append(ports, int64(servicePortWithoutSSL))
		}
		if servicePortWithSSL != 0 {
			ports = append(ports, int64(servicePortWithSSL))
		}
	}

	return ports
}

// default process names and ports as arrays for comparison
var processNames = []string{
	"beam.smp", "goxdcr", "memcached", "indexer", "projector", "cbq-engine",
	"js-evaluator", "cbft", "epmd", "saslauthd-port", "cbas", "eventing-produc",
	"backup",
}

// function to check if all couchbase ports are run by couchbase services only
func checkPortStatus(node *bootstrap.Node) *values.WrappedCheckerResult {
	result := &values.WrappedCheckerResult{
		Result: &values.CheckerResult{
			Name:   values.CheckPortStatus,
			Status: values.MissingCheckerStatus,
			Time:   time.Now().UTC(),
		},
		Cluster: node.RestClient().ClusterUUID(),
		Node:    node.UUID(),
	}

	ports := getAllPorts(node)

	procDir, err := os.Open("/proc")
	if err != nil {
		result.Error = fmt.Errorf("could not open /proc: %w", err)
		return result
	}
	defer procDir.Close()

	// STEP 1: checking proc/net/tcp for all port numbers and inode numbers.
	netFile, err := os.ReadFile("/proc/net/tcp")
	if err != nil {
		result.Error = fmt.Errorf("could not open /proc/net/tcp: %w", err)
		return result
	}

	portNumbers, inodeNumbers := getAllActivePortsAndInodeNumbers(netFile, ports)

	// STEP 2: checking through all PID folders and finding inode numbers for each PID.
	seenPIDs := make(map[int64]bool, len(portNumbers))
	for _, port := range portNumbers {
		seenPIDs[port] = false
	}

	blockedPorts := make([]int64, 0, len(portNumbers))
	blockedServiceName := make([]string, 0, len(portNumbers))

	dirs, err := procDir.Readdirnames(0)
	if err != nil {
		result.Error = fmt.Errorf("could not list processes: %w", err)
		return result
	}

	for _, dir := range dirs {
		pid, err := strconv.ParseInt(dir, 10, 0)
		if err != nil {
			continue // not a valid PID
		}

		pidDir, err := os.Open(fmt.Sprintf("/proc/%d/fd", pid))
		if errors.Is(err, os.ErrPermission) || errors.Is(err, os.ErrNotExist) {
			continue
		} else if err != nil {
			result.Error = fmt.Errorf("encountered error while opening /proc/%d/fd: %w", pid, err)
			return result
		}

		// Reading all files within a particular fd directory of a pid.
		fdDirs, err := pidDir.Readdirnames(0)
		if err != nil {
			result.Error = fmt.Errorf("could not read the fd directory for pid %d: %w", pid, err)
			return result
		}

		for _, fdDir := range fdDirs {
			fd, err := strconv.ParseInt(fdDir, 10, 0)
			if err != nil {
				continue // not a valid FD
			}

			readLink, err := os.Readlink(fmt.Sprintf("/proc/%d/fd/%d", pid, fd))
			if err != nil {
				zap.S().Warnw("(Port Status Check) Unable to read link",
					"file", fmt.Sprintf("/proc/%d/fd/%d", pid, fd), "error", err)
				continue
			}

			// obtained inode value from /proc/*/fd/* links
			var inodeObtained string
			// readLink example output: "socket:[4535]" -> extracting 4535
			if strings.Contains(readLink, "socket:") {
				inodeObtainedArr := strings.Split(readLink, ":")
				if len(inodeObtainedArr) > 1 {
					inodeObtained = inodeObtainedArr[1]
					inodeObtained = inodeObtained[1 : len(inodeObtained)-1]
				}
			}

			// if obtained inode is present in /proc/net/tcp -> find process name
			for i, inode := range inodeNumbers {
				if inodeObtained == inode {
					seenPIDs[portNumbers[i]] = true
					// STEP 3: compare process name with default couchbase process names
					foundProcessName := findProcessName(int(pid))
					if !slices.Contains(processNames, foundProcessName) {
						blockedPorts = append(blockedPorts, portNumbers[i])
						blockedServiceName = append(blockedServiceName, foundProcessName)
					}
				}
			}
		}
	}

	if len(blockedPorts) > 0 {
		result.Result.Status = values.WarnCheckerStatus
		result.Result.Remediation = fmt.Sprintf("The port(s) %v is being used by another "+
			"service and can't be used by couchbase. Please change the port for "+
			"the service(s): %v running on it.", blockedPorts, blockedServiceName)
		return result
	}

	// STEP 4: final check, if any couchbase port was not accessible by couchbase user.
	for port, value := range seenPIDs {
		if !value {
			blockedPorts = append(blockedPorts, port)
		}
	}

	if len(blockedPorts) > 0 {
		result.Result.Status = values.WarnCheckerStatus
		result.Result.Remediation = fmt.Sprintf("The port(s) %v is blocked and can't be used by "+
			"couchbase. Please change the port for the service(s) running on it.", blockedPorts)
		return result
	}

	result.Result.Status = values.GoodCheckerStatus
	return result
}

type AutoFailoverSettings struct {
	Timeout int `json:"timeout"`
}

const (
	HypervisorMatchPattern string = "hypervisor"
	AutoFailoverLowerLimit int    = 30
	AutoFailoverUpperLimit int    = 45
)

// checkForVM attempts to detect whether the machine this is running on is a virtual machine.
// Returns an error if it is unable to find out.
func checkForVM() (bool, error) {
	cmd := exec.Command("systemd-detect-virt")
	stdout, err := cmd.Output()
	if err != nil {
		zap.S().Warnw("(Check Auto Failover for VM) unable to run systemd-detect-virt", "error", err)
	} else {
		cpuInfo, err := os.ReadFile("/proc/cpuinfo")
		if err != nil {
			return false, fmt.Errorf("failed to determine VM: %w", err)
		}
		if strings.Contains(string(cpuInfo), HypervisorMatchPattern) {
			return true, nil
		}
	}
	if string(stdout) != "none" {
		return true, nil
	}

	return false, nil
}

// checkAutoFailoverForVM checks if a couchbase server running on hypervisor has auto-failover
// threshold greater than a required time
func checkAutoFailoverForVM(node *bootstrap.Node) *values.WrappedCheckerResult {
	result := &values.WrappedCheckerResult{
		Result: &values.CheckerResult{
			Name:   values.CheckAutoFailoverForVM,
			Status: values.GoodCheckerStatus,
			Time:   time.Now().UTC(),
		},
		Cluster: node.RestClient().ClusterUUID(),
		Node:    node.UUID(),
	}

	var isVirtual bool
	isVirtual, result.Error = checkForVM()

	if result.Error != nil {
		return result
	}

	if !isVirtual {
		return result
	}

	response, err := node.RestClient().Execute(&cbrest.Request{
		Method:             http.MethodGet,
		Endpoint:           "/settings/autoFailover",
		Service:            cbrest.ServiceManagement,
		ExpectedStatusCode: http.StatusOK,
	})
	if err != nil {
		result.Error = fmt.Errorf("failed to fetch /settings/autoFailover: %w", err)
		return result
	}

	var parsedResponse AutoFailoverSettings
	if err = json.Unmarshal(response.Body, &parsedResponse); err != nil {
		result.Error = fmt.Errorf("unable to unmarshal response from /settings/autoFailover: %w", err)
		return result
	}

	if parsedResponse.Timeout < AutoFailoverLowerLimit {
		result.Result.Status = values.WarnCheckerStatus
		result.Result.Remediation = fmt.Sprintf("Please increase the auto failover timeout to "+
			"%d or more, depending on your workload, as you are using a virtual machine.", AutoFailoverUpperLimit)
	} else if parsedResponse.Timeout >= AutoFailoverLowerLimit && parsedResponse.Timeout < AutoFailoverUpperLimit {
		result.Result.Status = values.InfoCheckerStatus
		result.Result.Remediation = fmt.Sprintf("Your auto failover timeout is %d. Please increase "+
			"to %d or more, depending on your workload, as you are using a virtual machine.",
			parsedResponse.Timeout, AutoFailoverUpperLimit)
	}

	return result
}
