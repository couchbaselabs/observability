package api

import (
	"fmt"
	"net/http"
	"os"

	"github.com/couchbaselabs/observability/config-svc/pkg/alertmanager"
	v1 "github.com/couchbaselabs/observability/config-svc/pkg/api/v1"
	"github.com/labstack/echo/v4"
	"gopkg.in/guregu/null.v4"
	"gopkg.in/yaml.v3"
)

const (
	defaultAlertmanagerConfigPath = "/etc/alertmanager/config.yml"
)

func (s *Server) GetAlertsConfiguration(ctx echo.Context) error {
	cfgPath := os.Getenv("ALERTMANAGER_CONFIG_FILE")
	if cfgPath == "" {
		cfgPath = defaultAlertmanagerConfigPath
	}
	fd, err := os.OpenFile(cfgPath, os.O_RDONLY, 0)
	if err != nil {
		return fmt.Errorf("failed to open alertmanager config: %w", err)
	}
	defer fd.Close()
	var cfg alertmanager.ConfigFile
	if err := yaml.NewDecoder(fd).Decode(&cfg); err != nil {
		return fmt.Errorf("failed to read alertmanager config: %w", err)
	}

	result := v1.AlertNotificationConfig{
		Email: nil,
		Slack: &v1.SlackAlertNotificationConfig{},
	}
	if cfg.Global.SlackAPIURLFile != "" {
		if stat, err := os.Stat(cfg.Global.SlackAPIURLFile); err == nil && stat.Size() > 0 {
			result.Slack.ConfiguredExternally = null.BoolFrom(true).Ptr()
		}
	}
	if cfg.Global.SlackAPIURL != "" {
		result.Slack.WebhookURL = cfg.Global.SlackAPIURL
	}

	result.Email = &v1.EmailAlertNotificationConfig{
		From:       cfg.Global.SMTPFrom,
		Host:       cfg.Global.SMTPSmarthost,
		Hello:      null.StringFrom(cfg.Global.SMTPHello).Ptr(),
		Identity:   null.StringFrom(cfg.Global.SMTPAuthIdentity).Ptr(),
		Password:   null.StringFrom(cfg.Global.SMTPAuthPassword).Ptr(),
		RequireTLS: null.BoolFrom(cfg.Global.SMTPRequireTLS).Ptr(),
		Secret:     null.StringFrom(cfg.Global.SMTPAuthSecret).Ptr(),
		Username:   null.StringFrom(cfg.Global.SMTPAuthUsername).Ptr(),
	}

	return ctx.JSON(http.StatusOK, result)
}

func (s *Server) PutAlertsConfiguration(ctx echo.Context) error {
	var payload v1.PutAlertsConfigurationJSONBody
	if err := ctx.Bind(&payload); err != nil {
		return err
	}

	cfgPath := os.Getenv("ALERTMANAGER_CONFIG_FILE")
	if cfgPath == "" {
		cfgPath = defaultAlertmanagerConfigPath
	}
	fd, err := os.OpenFile(cfgPath, os.O_RDWR, 0)
	if err != nil {
		return fmt.Errorf("failed to open alertmanager config: %w", err)
	}
	defer fd.Close()
	var cfg alertmanager.ConfigFile
	if err := yaml.NewDecoder(fd).Decode(&cfg); err != nil {
		return fmt.Errorf("failed to read alertmanager config: %w", err)
	}

	if payload.Email != nil {
		cfg.Global.SMTPFrom = payload.Email.From
		hello := null.StringFromPtr(payload.Email.Hello)
		if hello.Valid {
			cfg.Global.SMTPHello = hello.String
		}
		cfg.Global.SMTPSmarthost = payload.Email.Host
		cfg.Global.SMTPAuthUsername = null.StringFromPtr(payload.Email.Username).ValueOrZero()
		cfg.Global.SMTPAuthSecret = null.StringFromPtr(payload.Email.Secret).ValueOrZero()
		cfg.Global.SMTPAuthIdentity = null.StringFromPtr(payload.Email.Identity).ValueOrZero()
		cfg.Global.SMTPAuthPassword = null.StringFromPtr(payload.Email.Password).ValueOrZero()
		cfg.Global.SMTPRequireTLS = null.BoolFromPtr(payload.Email.RequireTLS).ValueOrZero()
	}

	if payload.Slack != nil {
		cfg.Global.SlackAPIURL = payload.Slack.WebhookURL
	}

	yamlVal, err := yaml.Marshal(&cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal AM config: %w", err)
	}
	if err := overwriteFileContents(fd, yamlVal); err != nil {
		return fmt.Errorf("failed to write AM config: %w", err)
	}

	return ctx.JSON(http.StatusOK, map[string]interface{}{"ok": true})
}
