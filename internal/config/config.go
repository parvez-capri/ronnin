package config

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
)

// Config represents the application configuration
type Config struct {
	Port               int      `mapstructure:"PORT" validate:"required,min=1024,max=65535"`
	Environment        string   `mapstructure:"ENV" validate:"required,oneof=development staging production"`
	LogLevel           string   `mapstructure:"LOG_LEVEL" validate:"required,oneof=debug info warn error"`
	CORSAllowedOrigins []string `mapstructure:"CORS_ALLOWED_ORIGINS" validate:"required,dive,url"`
	DatabaseURL        string   `mapstructure:"DATABASE_URL"`
	JiraURL            string   `mapstructure:"JIRA_URL" validate:"required,url"`
	JiraUsername       string   `mapstructure:"JIRA_USERNAME" validate:"required,email"`
	JiraAPIToken       string   `mapstructure:"JIRA_API_TOKEN" validate:"required"`
	JiraProjectKey     string   `mapstructure:"JIRA_PROJECT_KEY" validate:"required"`
	SupportTeamMembers []string `mapstructure:"SUPPORT_TEAM_MEMBERS" validate:"required,dive,min=1"`
	DefaultPriority    string   `mapstructure:"DEFAULT_PRIORITY" validate:"oneof=Highest High Medium Low Lowest"`

	// S3 Configuration
	AWSS3AccessKey  string `mapstructure:"AWS_S3_ACCESS_KEY"`
	AWSS3SecretKey  string `mapstructure:"AWS_S3_SECRET_KEY"`
	AWSS3Region     string `mapstructure:"AWS_S3_REGION" validate:"required_with=AWSS3AccessKey"`
	AWSS3BucketName string `mapstructure:"AWS_S3_BUCKET_NAME" validate:"required_with=AWSS3AccessKey"`
	AWSS3BaseURL    string `mapstructure:"AWS_S3_BASE_URL"`

	// MongoDB Configuration
	MongoURI        string `mapstructure:"MONGO_URI"`
	MongoDB         string `mapstructure:"MONGO_DB"`
	MongoCollection string `mapstructure:"MONGO_COLLECTION"`
}

func Load() (*Config, error) {
	// Set default values
	viper.SetDefault("PORT", 8080)
	viper.SetDefault("ENV", "development")
	viper.SetDefault("LOG_LEVEL", "info")
	viper.SetDefault("CORS_ALLOWED_ORIGINS", []string{"http://localhost:8080"})
	viper.SetDefault("ENVIRONMENT", "development")

	// Default MongoDB values for local development
	viper.SetDefault("MONGO_URI", "mongodb://localhost:27017")
	viper.SetDefault("MONGO_DB", "ronnin")
	viper.SetDefault("MONGO_COLLECTION", "tickets")

	// Configure viper
	viper.SetConfigFile(".env")
	viper.SetConfigType("env")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Read config file
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Handle CORS_ALLOWED_ORIGINS as comma-separated string
	if corsOrigins := viper.GetString("CORS_ALLOWED_ORIGINS"); corsOrigins != "" {
		cfg.CORSAllowedOrigins = strings.Split(corsOrigins, ",")
	}

	// Handle SUPPORT_TEAM_MEMBERS as comma-separated string
	if teamMembers := viper.GetString("SUPPORT_TEAM_MEMBERS"); teamMembers != "" {
		cfg.SupportTeamMembers = strings.Split(teamMembers, ",")
	}

	// Validate config
	validate := validator.New()
	if err := validate.Struct(&cfg); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	return &cfg, nil
}
