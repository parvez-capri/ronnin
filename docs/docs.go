// Package docs Code generated by swaggo/swag. DO NOT EDIT
package docs

import "github.com/swaggo/swag"

const docTemplate = `{
    "schemes": {{ marshal .Schemes }},
    "swagger": "2.0",
    "info": {
        "description": "{{escape .Description}}",
        "title": "{{.Title}}",
        "termsOfService": "http://swagger.io/terms/",
        "contact": {
            "name": "API Support",
            "url": "http://www.swagger.io/support",
            "email": "support@swagger.io"
        },
        "license": {
            "name": "Apache 2.0",
            "url": "http://www.apache.org/licenses/LICENSE-2.0.html"
        },
        "version": "{{.Version}}"
    },
    "host": "{{.Host}}",
    "basePath": "{{.BasePath}}",
    "paths": {
        "/create-ticket": {
            "post": {
                "description": "Creates a new JIRA ticket with the provided information",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "tickets"
                ],
                "summary": "Create a new ticket",
                "parameters": [
                    {
                        "description": "Ticket creation request",
                        "name": "request",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "$ref": "#/definitions/github_com_parvez-capri_ronnin_internal_models.TicketRequest"
                        }
                    }
                ],
                "responses": {
                    "201": {
                        "description": "Created",
                        "schema": {
                            "$ref": "#/definitions/github_com_parvez-capri_ronnin_internal_models.TicketResponse"
                        }
                    },
                    "400": {
                        "description": "Invalid request body or validation failed",
                        "schema": {
                            "$ref": "#/definitions/github_com_parvez-capri_ronnin_internal_models.ErrorResponse"
                        }
                    },
                    "500": {
                        "description": "Internal server error",
                        "schema": {
                            "$ref": "#/definitions/github_com_parvez-capri_ronnin_internal_models.ErrorResponse"
                        }
                    }
                }
            }
        },
        "/health": {
            "get": {
                "description": "Get the status of server and its dependencies",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "health"
                ],
                "summary": "Health check endpoint",
                "responses": {
                    "200": {
                        "description": "OK",
                        "schema": {
                            "$ref": "#/definitions/github_com_parvez-capri_ronnin_internal_models.HealthResponse"
                        }
                    },
                    "503": {
                        "description": "Service Unavailable",
                        "schema": {
                            "$ref": "#/definitions/github_com_parvez-capri_ronnin_internal_models.ErrorResponse"
                        }
                    }
                }
            }
        }
    },
    "definitions": {
        "github_com_parvez-capri_ronnin_internal_models.ErrorResponse": {
            "type": "object",
            "properties": {
                "details": {
                    "type": "string",
                    "example": "Field 'url' is required"
                },
                "error": {
                    "type": "string",
                    "example": "Invalid request body"
                }
            }
        },
        "github_com_parvez-capri_ronnin_internal_models.HealthResponse": {
            "type": "object",
            "properties": {
                "services": {
                    "type": "object",
                    "additionalProperties": {
                        "type": "string"
                    }
                },
                "status": {
                    "type": "string",
                    "example": "ok"
                },
                "timestamp": {
                    "type": "integer",
                    "example": 1647123456
                }
            }
        },
        "github_com_parvez-capri_ronnin_internal_models.TicketRequest": {
            "type": "object",
            "required": [
                "imageS3URL",
                "payload",
                "requestHeaders",
                "response",
                "url"
            ],
            "properties": {
                "imageS3URL": {
                    "type": "string",
                    "example": "https://bucket.s3.amazonaws.com/screenshot.png"
                },
                "payload": {
                    "type": "object",
                    "additionalProperties": true
                },
                "requestHeaders": {
                    "type": "object",
                    "additionalProperties": {
                        "type": "string"
                    }
                },
                "response": {
                    "type": "object",
                    "additionalProperties": true
                },
                "url": {
                    "type": "string",
                    "example": "https://example.com/api/endpoint"
                }
            }
        },
        "github_com_parvez-capri_ronnin_internal_models.TicketResponse": {
            "type": "object",
            "properties": {
                "assignedTo": {
                    "type": "string",
                    "example": "john.doe@company.com"
                },
                "jiraLink": {
                    "type": "string",
                    "example": "https://your-jira.atlassian.net/browse/PROJECT-123"
                },
                "status": {
                    "type": "string",
                    "example": "created"
                },
                "ticketId": {
                    "type": "string",
                    "example": "PROJECT-123"
                }
            }
        }
    },
    "securityDefinitions": {
        "ApiKeyAuth": {
            "type": "apiKey",
            "name": "Authorization",
            "in": "header"
        }
    },
    "x-extension-openapi": {
        "example": "value on a json format"
    }
}`

// SwaggerInfo holds exported Swagger Info so clients can modify it
var SwaggerInfo = &swag.Spec{
	Version:          "1.0",
	Host:             "localhost:8080",
	BasePath:         "/",
	Schemes:          []string{},
	Title:            "Ronnin API",
	Description:      "API Server for Ronnin application",
	InfoInstanceName: "swagger",
	SwaggerTemplate:  docTemplate,
	LeftDelim:        "{{",
	RightDelim:       "}}",
}

func init() {
	swag.Register(SwaggerInfo.InstanceName(), SwaggerInfo)
}
