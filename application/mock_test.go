package application

import (
	"context"
	"time"

	"github.com/renesul/ok/domain"
)

// --- Mock ConversationRepository ---

type mockConversationRepo struct {
	createCalled bool
	lastConv     *domain.Conversation
	createErr    error
	findByIDResult *domain.Conversation
	findByIDErr    error
	findAllResult  []domain.Conversation
	updateCalled   bool
	deleteCalled   bool
	searchResult   []domain.Conversation
}

func (r *mockConversationRepo) Create(_ context.Context, c *domain.Conversation) error {
	r.createCalled = true
	r.lastConv = c
	if r.createErr == nil {
		c.ID = 1
	}
	return r.createErr
}

func (r *mockConversationRepo) FindByID(_ context.Context, id uint) (*domain.Conversation, error) {
	return r.findByIDResult, r.findByIDErr
}

func (r *mockConversationRepo) FindAll(_ context.Context) ([]domain.Conversation, error) {
	return r.findAllResult, nil
}

func (r *mockConversationRepo) Update(_ context.Context, c *domain.Conversation) error {
	r.updateCalled = true
	return nil
}

func (r *mockConversationRepo) Delete(_ context.Context, id uint) error {
	r.deleteCalled = true
	return nil
}

func (r *mockConversationRepo) Search(_ context.Context, query string) ([]domain.Conversation, error) {
	return r.searchResult, nil
}

// --- Mock MessageRepository ---

type mockMessageRepo struct {
	createCalled      bool
	createBatchCalled bool
	lastMessages      []domain.Message
	findResult        []domain.Message
	countResult       int64
	indexCalled       bool
	deleteMsgCalled   bool
	deleteFTSCalled   bool
	deleteEmbCalled   bool
}

func (r *mockMessageRepo) Create(_ context.Context, m *domain.Message) error {
	r.createCalled = true
	m.ID = 1
	return nil
}

func (r *mockMessageRepo) CreateBatch(_ context.Context, msgs []domain.Message) error {
	r.createBatchCalled = true
	r.lastMessages = msgs
	for i := range msgs {
		msgs[i].ID = uint(i + 1)
	}
	return nil
}

func (r *mockMessageRepo) FindByConversationID(_ context.Context, _ uint) ([]domain.Message, error) {
	return r.findResult, nil
}

func (r *mockMessageRepo) CountByConversationID(_ context.Context, _ uint) (int64, error) {
	return r.countResult, nil
}

func (r *mockMessageRepo) IndexForSearch(_ context.Context, _ []domain.Message) error {
	r.indexCalled = true
	return nil
}

func (r *mockMessageRepo) SaveEmbedding(_ context.Context, _ uint, _ uint, _ []float32) error {
	return nil
}

func (r *mockMessageRepo) FindAllEmbeddings(_ context.Context) ([]domain.MessageEmbedding, error) {
	return nil, nil
}

func (r *mockMessageRepo) DeleteByConversationID(_ context.Context, _ uint) error {
	r.deleteMsgCalled = true
	return nil
}

func (r *mockMessageRepo) DeleteSearchIndex(_ context.Context, _ uint) error {
	r.deleteFTSCalled = true
	return nil
}

func (r *mockMessageRepo) DeleteEmbeddings(_ context.Context, _ uint) error {
	r.deleteEmbCalled = true
	return nil
}

// --- Mock JobRepository ---

type mockJobRepo struct {
	createCalled  bool
	updateCalled  bool
	deleteCalled  bool
	lastJob       *domain.Job
	createErr     error
	findByIDResult *domain.Job
}

func (r *mockJobRepo) Create(_ context.Context, j *domain.Job) error {
	r.createCalled = true
	r.lastJob = j
	return r.createErr
}

func (r *mockJobRepo) FindAll(_ context.Context) ([]domain.Job, error) {
	return nil, nil
}

func (r *mockJobRepo) FindByID(_ context.Context, _ string) (*domain.Job, error) {
	return r.findByIDResult, nil
}

func (r *mockJobRepo) Update(_ context.Context, j *domain.Job) error {
	r.updateCalled = true
	r.lastJob = j
	return nil
}

func (r *mockJobRepo) Delete(_ context.Context, _ string) error {
	r.deleteCalled = true
	return nil
}

// --- Mock SessionRepository ---

type mockSessionRepo struct {
	createCalled   bool
	findResult     *domain.Session
	findErr        error
	deleteCalled   bool
	deleteExpCalled bool
}

func (r *mockSessionRepo) Create(_ context.Context, s *domain.Session) error {
	r.createCalled = true
	return nil
}

func (r *mockSessionRepo) FindByID(_ context.Context, _ string) (*domain.Session, error) {
	return r.findResult, r.findErr
}

func (r *mockSessionRepo) DeleteByID(_ context.Context, _ string) error {
	r.deleteCalled = true
	return nil
}

func (r *mockSessionRepo) DeleteExpired(_ context.Context) error {
	r.deleteExpCalled = true
	return nil
}

// helper
func timePtr(t time.Time) time.Time { return t }
