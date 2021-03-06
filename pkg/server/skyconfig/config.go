// Copyright 2015-present Oursky Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package skyconfig

import (
	"errors"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
	"github.com/skygeario/skygear-server/pkg/server/uuid"
)

// parseBool parses a string representation of boolean value into a boolean
// type. In addition to what strconv.ParseBool recognize, it also recognize
// Yes, yes, YES, y, No, no, NO, n. Any other value returns an error.
func parseBool(str string) (bool, error) {
	switch str {
	case "Yes", "yes", "YES", "y":
		return true, nil
	case "No", "no", "NO", "n":
		return false, nil
	default:
		return strconv.ParseBool(str)
	}
}

type PluginConfig struct {
	Transport string
	Path      string
	Args      []string
}

// Configuration is Skygear's configuration
// The configuration will load in following order:
// 1. The ENV
// 2. The key/value in .env file
// 3. The config in *.ini (To-be depreacted)
type Configuration struct {
	HTTP struct {
		Host string `json:"host"`
	} `json:"http"`
	App struct {
		Name          string `json:"name"`
		APIKey        string `json:"api_key"`
		MasterKey     string `json:"master_key"`
		AccessControl string `json:"access_control"`
		DevMode       bool   `json:"dev_mode"`
		CORSHost      string `json:"cors_host"`
		Slave         bool   `json:"slave"`
	} `json:"app"`
	DB struct {
		ImplName string `json:"implementation"`
		Option   string `json:"option"`
	} `json:"database"`
	TokenStore struct {
		ImplName string `json:"implementation"`
		Path     string `json:"path"`
		Prefix   string `json:"prefix"`
		Expiry   int64  `json:"expiry"`
		Secret   string `json:"secret"`
	} `json:"-"`
	AssetStore struct {
		ImplName string `json:"implementation"`
		Public   bool   `json:"public"`

		// followings only used when ImplName = fs
		Path string `json:"-"`

		// followings only used when ImplName = s3
		AccessToken string `json:"access_key"`
		SecretToken string `json:"secret_key"`
		Region      string `json:"region"`
		Bucket      string `json:"bucket"`

		// followings only used when ImplName = cloud
		CloudAssetHost          string `json:"cloud_asset_host"`
		CloudAssetToken         string `json:"cloud_asset_token"`
		CloudAssetPublicPrefix  string `json:"cloud_asset_public_prefix"`
		CloudAssetPrivatePrefix string `json:"cloud_asset_private_prefix"`
	} `json:"asset_store"`
	AssetURLSigner struct {
		URLPrefix string `json:"url_prefix"`
		Secret    string `json:"secret"`
	} `json:"asset_signer"`
	APNS struct {
		Enable   bool   `json:"enable"`
		Env      string `json:"env"`
		Cert     string `json:"cert"`
		Key      string `json:"key"`
		CertPath string `json:"-"`
		KeyPath  string `json:"-"`
	} `json:"apns"`
	GCM struct {
		Enable bool   `json:"enable"`
		APIKey string `json:"api_key"`
	} `json:"gcm"`
	LOG struct {
		Level        string            `json:"-"`
		LoggersLevel map[string]string `json:"-"`
	} `json:"log"`
	LogHook struct {
		SentryDSN   string
		SentryLevel string
	} `json:"-"`
	Plugin map[string]*PluginConfig `json:"-"`
}

func NewConfiguration() Configuration {
	config := Configuration{}
	config.HTTP.Host = ":3000"
	config.App.Name = "myapp"
	config.App.AccessControl = "role"
	config.App.DevMode = true
	config.App.CORSHost = "*"
	config.App.Slave = false
	config.DB.ImplName = "pq"
	config.DB.Option = "postgres://postgres:@localhost/postgres?sslmode=disable"
	config.TokenStore.ImplName = "fs"
	config.TokenStore.Path = "data/token"
	config.TokenStore.Expiry = 0
	config.AssetStore.ImplName = "fs"
	config.AssetStore.Path = "data/asset"
	config.AssetURLSigner.URLPrefix = "http://localhost:3000/files"
	config.APNS.Enable = false
	config.APNS.Env = "sandbox"
	config.GCM.Enable = false
	config.LOG.Level = "debug"
	config.LOG.LoggersLevel = map[string]string{
		"plugin": "info",
	}
	config.Plugin = map[string]*PluginConfig{}
	return config
}

func NewConfigurationWithKeys() Configuration {
	config := NewConfiguration()
	config.App.APIKey = uuid.New()
	config.App.MasterKey = uuid.New()
	return config
}

func (config *Configuration) Validate() error {
	if config.App.Name == "" {
		return errors.New("APP_NAME is not set")
	}
	if config.App.APIKey == "" {
		return errors.New("API_KEY is not set")
	}
	if config.App.MasterKey == "" {
		return errors.New("MASTER_KEY is not set")
	}
	if !regexp.MustCompile("^[A-Za-z0-9_]+$").MatchString(config.App.Name) {
		return fmt.Errorf("APP_NAME '%s' contains invalid characters other than alphanumerics or underscores", config.App.Name)
	}
	if config.APNS.Enable && !regexp.MustCompile("^(sandbox|production)$").MatchString(config.APNS.Env) {
		return fmt.Errorf("APNS_ENV must be sandbox or production")
	}
	return nil
}

func (config *Configuration) ReadFromEnv() {
	envErr := godotenv.Load()
	if envErr != nil {
		log.Print("Error in loading .env file")
	}

	config.readHost()

	appAPIKey := os.Getenv("API_KEY")
	if appAPIKey != "" {
		config.App.APIKey = appAPIKey
	}

	appMasterKey := os.Getenv("MASTER_KEY")
	if appMasterKey != "" {
		config.App.MasterKey = appMasterKey
	}

	appName := os.Getenv("APP_NAME")
	if appName != "" {
		config.App.Name = appName
	}

	corsHost := os.Getenv("CORS_HOST")
	if corsHost != "" {
		config.App.CORSHost = corsHost
	}

	accessControl := os.Getenv("ACCESS_CONRTOL")
	if accessControl != "" {
		config.App.AccessControl = accessControl
	}

	if devMode, err := parseBool(os.Getenv("DEV_MODE")); err == nil {
		config.App.DevMode = devMode
	}

	dbImplName := os.Getenv("DB_IMPL_NAME")
	if dbImplName != "" {
		config.DB.ImplName = dbImplName
	}

	if config.DB.ImplName == "pq" && os.Getenv("DATABASE_URL") != "" {
		config.DB.Option = os.Getenv("DATABASE_URL")
	}

	if slave, err := parseBool(os.Getenv("SLAVE")); err == nil {
		config.App.Slave = slave
	}

	config.readTokenStore()
	config.readAssetStore()
	config.readAPNS()
	config.readGCM()
	config.readLog()
	config.readPlugins()
}

func (config *Configuration) readHost() {
	// Default to :3000 if both HOST and PORT is missing
	host := os.Getenv("HOST")
	if host != "" {
		config.HTTP.Host = host
	}
	if config.HTTP.Host == "" {
		port := os.Getenv("PORT")
		if port != "" {
			config.HTTP.Host = ":" + port
		}
	}
}

func (config *Configuration) readTokenStore() {
	tokenStore := os.Getenv("TOKEN_STORE")
	if tokenStore != "" {
		config.TokenStore.ImplName = tokenStore
	}
	tokenStorePath := os.Getenv("TOKEN_STORE_PATH")
	if tokenStorePath != "" {
		config.TokenStore.Path = tokenStorePath
	}

	tokenStorePrefix := os.Getenv("TOKEN_STORE_PREFIX")
	if tokenStorePrefix != "" {
		config.TokenStore.Prefix = tokenStorePrefix
	}

	if expiry, err := strconv.ParseInt(os.Getenv("TOKEN_STORE_EXPIRY"), 10, 64); err == nil {
		config.TokenStore.Expiry = expiry
	}

	tokenStoreSecret := os.Getenv("TOKEN_STORE_SECRET")
	if tokenStoreSecret != "" {
		config.TokenStore.Secret = tokenStoreSecret
	} else {
		config.TokenStore.Secret = config.App.MasterKey
	}
}

func (config *Configuration) readAssetStore() {
	assetStore := os.Getenv("ASSET_STORE")
	if assetStore != "" {
		config.AssetStore.ImplName = assetStore
	}

	if assetStorePublic, err := parseBool(os.Getenv("ASSET_STORE_PUBLIC")); err == nil {
		config.AssetStore.Public = assetStorePublic
	}

	// Local Storage related
	assetStorePath := os.Getenv("ASSET_STORE_PATH")
	if assetStorePath != "" {
		config.AssetStore.Path = assetStorePath
	}
	assetStorePrefix := os.Getenv("ASSET_STORE_URL_PREFIX")
	if assetStorePrefix != "" {
		config.AssetURLSigner.URLPrefix = assetStorePrefix
	}
	assetStoreSecret := os.Getenv("ASSET_STORE_SECRET")
	if assetStoreSecret != "" {
		config.AssetURLSigner.Secret = assetStoreSecret
	}

	// S3 related
	assetStoreAccessKey := os.Getenv("ASSET_STORE_ACCESS_KEY")
	if assetStoreAccessKey != "" {
		config.AssetStore.AccessToken = assetStoreAccessKey
	}
	assetStoreSecretKey := os.Getenv("ASSET_STORE_SECRET_KEY")
	if assetStoreSecretKey != "" {
		config.AssetStore.SecretToken = assetStoreSecretKey
	}
	assetStoreRegion := os.Getenv("ASSET_STORE_REGION")
	if assetStoreRegion != "" {
		config.AssetStore.Region = assetStoreRegion
	}
	assetStoreBucket := os.Getenv("ASSET_STORE_BUCKET")
	if assetStoreBucket != "" {
		config.AssetStore.Bucket = assetStoreBucket
	}

	// Cloud Asset related
	cloudAssetHost := os.Getenv("CLOUD_ASSET_HOST")
	if cloudAssetHost != "" {
		config.AssetStore.CloudAssetHost = cloudAssetHost
	}
	cloudAssetToken := os.Getenv("CLOUD_ASSET_TOKEN")
	if cloudAssetToken != "" {
		config.AssetStore.CloudAssetToken = cloudAssetToken
	}
	cloudAssetPublicPrefix := os.Getenv("CLOUD_ASSET_PUBLIC_PREFIX")
	if cloudAssetPublicPrefix != "" {
		config.AssetStore.CloudAssetPublicPrefix = cloudAssetPublicPrefix
	}
	cloudAssetPrivatePrefix := os.Getenv("CLOUD_ASSET_PRIVATE_PREFIX")
	if cloudAssetPrivatePrefix != "" {
		config.AssetStore.CloudAssetPrivatePrefix = cloudAssetPrivatePrefix
	}
}

func (config *Configuration) readAPNS() {
	if shouldEnableAPNS, err := parseBool(os.Getenv("APNS_ENABLE")); err == nil {
		config.APNS.Enable = shouldEnableAPNS
	}

	if !config.APNS.Enable {
		return
	}

	env := os.Getenv("APNS_ENV")
	if env != "" {
		config.APNS.Env = env
	}

	cert, key := os.Getenv("APNS_CERTIFICATE"), os.Getenv("APNS_PRIVATE_KEY")
	if cert != "" {
		config.APNS.Cert = cert
	}
	if key != "" {
		config.APNS.Key = key
	}

	certPath, keyPath := os.Getenv("APNS_CERTIFICATE_PATH"), os.Getenv("APNS_PRIVATE_KEY_PATH")
	if certPath != "" {
		config.APNS.CertPath = certPath
	}
	if keyPath != "" {
		config.APNS.KeyPath = keyPath
	}

}

func (config *Configuration) readGCM() {
	if shouldEnableGCM, err := parseBool(os.Getenv("GCM_ENABLE")); err == nil {
		config.GCM.Enable = shouldEnableGCM
	}

	gcmAPIKey := os.Getenv("GCM_APIKEY")
	if gcmAPIKey != "" {
		config.GCM.APIKey = gcmAPIKey
	}
}

func (config *Configuration) readLog() {
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel != "" {
		config.LOG.Level = logLevel
	}

	for _, environ := range os.Environ() {
		if !strings.HasPrefix(environ, "LOG_LEVEL_") {
			continue
		}

		components := strings.SplitN(environ, "=", 2)
		loggerName := strings.ToLower(strings.TrimPrefix(components[0], "LOG_LEVEL_"))
		loggerLevel := components[1]
		config.LOG.LoggersLevel[loggerName] = loggerLevel
	}

	sentry := os.Getenv("SENTRY_DSN")
	if sentry != "" {
		config.LogHook.SentryDSN = sentry
	}

	sentryLevel := os.Getenv("SENTRY_LEVEL")
	if sentryLevel != "" {
		config.LogHook.SentryLevel = sentryLevel
	}
}

func (config *Configuration) readPlugins() {
	plugin := os.Getenv("PLUGINS")
	if plugin == "" {
		return
	}

	plugins := strings.Split(plugin, ",")
	for _, p := range plugins {
		pluginConfig := &PluginConfig{}
		pluginConfig.Transport = os.Getenv(p + "_TRANSPORT")
		pluginConfig.Path = os.Getenv(p + "_PATH")
		args := os.Getenv(p + "_ARGS")
		if args != "" {
			pluginConfig.Args = strings.Split(args, ",")
		}
		config.Plugin[p] = pluginConfig
	}
}
