// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package parsers

import (
	"errors"
	"regexp"
	"strings"

	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/eventlog/values"
)

var (
	groupNameRegexp  = regexp.MustCompile(`{group_name,\<\<"(?P<group>[^"]*)"`)
	groupsRegexp     = regexp.MustCompile(`\{groups,\[(?P<groups>[^\]]*)\]`)
	TLSVersionRegexp = regexp.MustCompile(`'(?P<version>tlsv[^']*)'`)
)

// PasswordPolicyOrLDAPSettingsModified gets when the LDAP settings or the password policy are changed.
// Example ldap line: [ns_server:debug,2021-03-03T15:35:04.588Z,ns_1@10.144.210.101:ns_config_log<0.221.0>:ns_config_log
//
//	:log_common:232]config change:ldap_settings ->[{'_vclock',[{<<"21d355ae87f2145934a2429972e3cc7e">>,{1,
//	63782004904}}]},{hosts,[]},{port,389},{encryption,'None'},{server_cert_validation,true},{bind_dn,[]},{bind
//	_pass,{password,{sanitized,<<"Qwm8ulZt3keBHEf6F+2oJRdhkJo5k0065Iy1+y+QyxQ=">>}}},{client_tls_cert,undefined
//	},{client_tls_key,undefined},{cacert,undefined},{authentication_enabled,false},{authorization_enabled,true}
//	,{nested_groups_enabled,false},{groups_query,"%D?test-group?base"},{request_timeout,5000},{max_parallel_
//	connections,100},{max_cache_size,10000},{cache_value_lifetime,300000},{nested_groups_max_depth,10}]
//
// Example password policy line: [ns_server:debug,2021-03-05T10:24:59.025Z,ns_1@10.144.210.101:ns_config_log<0.221.0>:
//
//	ns_config_log:log_common:232]config change:password_policy ->[{min_length,6},{must_present,[]}]
func PasswordPolicyOrLDAPSettingsModified(line string) (*values.Result, error) {
	if !strings.Contains(line, "config change") ||
		(!strings.Contains(line, "ldap_settings") && !strings.Contains(line, "password_policy")) {
		return nil, values.ErrNotInLine
	}

	event := values.PasswordPolicyChangedEvent
	if strings.Contains(line, "ldap_settings") {
		event = values.LDAPSettingsModifiedEvent
	}

	return &values.Result{
		Event: event,
	}, nil
}

// GroupAddedOrRemoved gets when a user group is added to or removed from the cluster.
// Example added line: [ns_server:debug,2021-03-03T15:33:07.981Z,ns_1@10.144.210.101:ns_audit<0.519.0>:ns_audit:handle
//
//	_call:131]Audit set_user_group: [{reason,added},{description,<<>>},{ldap_group_ref,<<>>},{roles,[<<"admin">>]},
//	{group_name,<<"test-group">>},{real_userid,{[{domain,builtin},{user,<<"<ud>Administrator</ud>">>}]}},
//	{sessionid,<<"4ea6145d147dd995da695b6f5b68d35a">>},{local,{[{ip,<<"10.144.210.101">>},{port,8091}]}},
//	{remote,{[{ip,<<"10.144.210.1">>},{port,59607}]}},{timestamp,<<"2021-03-03T15:33:07.978Z">>}]
//
// Example removed line: [ns_server:debug,2021-03-03T15:33:38.325Z,ns_1@10.144.210.101:ns_audit<0.519.0>:ns_audit:handle
//
//	_call:131]Audit delete_user_group: [{group_name,<<"delete-group">>},{real_userid,{[{domain,builtin},{user,<<"
//	<ud>Administrator</ud>">>}]}},{sessionid,<<"4ea6145d147dd995da695b6f5b68d35a">>},{local,{[{ip,<<"10.144
//	.210.101">>},{port,8091}]}},{remote,{[{ip,<<"10.144.210.1">>},{port,59618}]}},{timestamp,<<"2021-03-03T15
//	:33:38.325Z">>}]
func GroupAddedOrRemoved(line string) (*values.Result, error) {
	output, err := getCaptureGroups(line, []string{"_user_group"}, []string{"reason,added", "delete_user_group"},
		groupNameRegexp, 2)
	if err != nil {
		return nil, err
	}

	event := values.GroupAddedEvent
	if strings.Contains(line, "delete_user_group") {
		event = values.GroupDeletedEvent
	}

	return &values.Result{
		Event: event,
		Group: output[1],
	}, nil
}

// UserAdded gets when a user is added to the cluster.
// Example Line: [ns_server:debug,2021-03-03T15:35:44.126Z,ns_1@10.144.210.101:ns_audit<0.519.0>:ns_audit:handle_call:
//
//	131]Audit set_user: [{reason,added},{groups,[<<"test-group">>]},{full_name,<<"<ud></ud>">>},{roles,[<
//	<"admin">>]},{identity,{[{domain,local},{user,<<"<ud>user-test</ud>">>}]}},{real_userid,{[{domain,builtin}
//	,{user,<<"<ud>Administrator</ud>">>}]}},{sessionid,<<"5726dbac66e87b61350d77205608b414">>},{local,{[{ip,<
//	<"10.144.210.101">>},{port,8091}]}},{remote,{[{ip,<<"10.144.210.1">>},{port,59641}]}},{timestamp,
//	<<"2021-03-03T15:35:44.125Z">>}]
func UserAdded(line string) (*values.Result, error) {
	output, err := getCaptureGroups(line, []string{"set_user:", "reason,added"}, nil, groupsRegexp, 2)
	if err != nil {
		return nil, err
	}

	groups := output[1]
	var groupList []string
	if groups != "" {
		groupList = strings.Split(groups, ",")
	}

	for i, group := range groupList {
		if len(group) < 6 {
			return nil, errors.New("groups contains group in unexpected format")
		}

		groupList[i] = group[3 : len(group)-3]
	}

	return &values.Result{
		Event:  values.UserAddedEvent,
		Groups: groupList,
	}, nil
}

// UserRemoved gets when a user is removed from the cluster.
// Example line: [ns_server:debug,2021-03-10T16:59:29.190Z,ns_1@10.144.210.101:ns_audit<0.1537.1216>:ns_audit:handle
//
//	_call:131]Audit delete_user: [{identity,{[{domain,local},{user,<<"<ud>user1</ud>">>}]}},{real_userid,{[
//	{domain,builtin},{user,<<"<ud>Administrator</ud>">>}]}},{sessionid,<<"c46b7512c1f18de6f390665e0df999c3">>}
//	,{local,{[{ip,<<"10.144.210.101">>},{port,8091}]}},{remote,{[{ip,<<"10.144.210.1">>},{port,54357}]}},
//	{timestamp,<<"2021-03-10T16:59:29.183Z">>}]
func UserRemoved(line string) (*values.Result, error) {
	if !strings.Contains(line, "delete_user:") {
		return nil, values.ErrNotInLine
	}

	return &values.Result{
		Event: values.UserDeletedEvent,
	}, nil
}

// MinTLSChanged gets when the minTLS is changed.
// Example line: [ns_server:debug,2021-04-09T08:57:48.749Z,ns_1@10.144.210.101:ns_config_log<0.223.0>:ns_config_log:
//
//	log_common:232]config change:ssl_minimum_protocol ->[{'_vclock',[{<<"21d355ae87f2145934a2429972e3cc7e">>,
//	{1,63785177868}}]}|'tlsv1.1']
func MinTLSChanged(line string) (*values.Result, error) {
	output, err := getCaptureGroups(line, []string{"ssl_minimum_protocol", "tls"}, nil, TLSVersionRegexp, 2)
	if err != nil {
		return nil, err
	}

	return &values.Result{
		Event:   values.MinTLSChangedEvent,
		Version: output[1],
	}, nil
}
