package services

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// FlattenedTicket represents a flattened version of ticket data for MongoDB storage
type FlattenedTicket struct {
	ID         primitive.ObjectID `bson:"_id,omitempty"`
	TicketID   string             `bson:"ticket_id"`
	Status     string             `bson:"status"`
	AssignedTo string             `bson:"assigned_to"`
	JiraLink   string             `bson:"jira_link"`
	CreatedAt  time.Time          `bson:"created_at"`

	// Issue details
	Issue       string `bson:"issue"`
	Description string `bson:"description"`
	UserEmail   string `bson:"user_email"`
	LeadID      string `bson:"lead_id"`
	Product     string `bson:"product"`
	PageURL     string `bson:"page_url"`
	ImageURL    string `bson:"image_url"`

	// Store JSON strings for complex data
	FailedNetworkCallsJSON string `bson:"failed_network_calls_json"`
	PayloadJSON            string `bson:"payload_json"`
	ResponseJSON           string `bson:"response_json"`
	RequestHeadersJSON     string `bson:"request_headers_json"`
}

// MongoDBService handles database operations
type MongoDBService struct {
	client     *mongo.Client
	database   *mongo.Database
	collection *mongo.Collection
}

// NewMongoDBService creates a new MongoDB service
func NewMongoDBService(uri, dbName, collectionName string) (*MongoDBService, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Connect to MongoDB
	clientOptions := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Ping the MongoDB server to verify connection
	err = client.Ping(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	// Get database and collection
	database := client.Database(dbName)
	collection := database.Collection(collectionName)

	return &MongoDBService{
		client:     client,
		database:   database,
		collection: collection,
	}, nil
}

// SaveTicket saves a ticket to MongoDB
func (s *MongoDBService) SaveTicket(ctx context.Context, ticket *FlattenedTicket) (string, error) {
	// Set creation time if not already set
	if ticket.CreatedAt.IsZero() {
		ticket.CreatedAt = time.Now()
	}

	// Insert the ticket
	result, err := s.collection.InsertOne(ctx, ticket)
	if err != nil {
		return "", fmt.Errorf("failed to insert ticket: %w", err)
	}

	// Return the ID of the inserted document
	if id, ok := result.InsertedID.(primitive.ObjectID); ok {
		return id.Hex(), nil
	}

	return "", fmt.Errorf("failed to get inserted ID")
}

// GetTicketByJiraID retrieves a ticket by its Jira ID
func (s *MongoDBService) GetTicketByJiraID(ctx context.Context, jiraID string) (*FlattenedTicket, error) {
	var ticket FlattenedTicket

	filter := bson.M{"ticket_id": jiraID}
	err := s.collection.FindOne(ctx, filter).Decode(&ticket)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("ticket not found: %s", jiraID)
		}
		return nil, fmt.Errorf("failed to get ticket: %w", err)
	}

	return &ticket, nil
}

// GetAllTickets retrieves all tickets
func (s *MongoDBService) GetAllTickets(ctx context.Context) ([]FlattenedTicket, error) {
	var tickets []FlattenedTicket

	cursor, err := s.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("failed to find tickets: %w", err)
	}
	defer cursor.Close(ctx)

	if err = cursor.All(ctx, &tickets); err != nil {
		return nil, fmt.Errorf("failed to decode tickets: %w", err)
	}

	return tickets, nil
}

// Disconnect closes the MongoDB connection
func (s *MongoDBService) Disconnect(ctx context.Context) error {
	return s.client.Disconnect(ctx)
}
