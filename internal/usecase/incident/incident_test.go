package incident

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/sday-kenta/backend/internal/entity"
	"github.com/sday-kenta/backend/internal/incidenterr"
)

func TestNormalizeCreateStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		input  string
		want   string
		hasErr bool
	}{
		{name: "empty becomes review", input: "", want: entity.IncidentStatusReview},
		{name: "draft stays draft", input: entity.IncidentStatusDraft, want: entity.IncidentStatusDraft},
		{name: "review stays review", input: entity.IncidentStatusReview, want: entity.IncidentStatusReview},
		{name: "published becomes review", input: entity.IncidentStatusPublished, want: entity.IncidentStatusReview},
		{name: "invalid rejected", input: "unknown", hasErr: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := normalizeCreateStatus(tt.input)
			if tt.hasErr {
				require.ErrorIs(t, err, incidenterr.ErrInvalidStatus)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestUpdateNonAdminCannotPublishIncident(t *testing.T) {
	t.Parallel()

	repo := &incidentRepoStub{
		incident: entity.Incident{
			ID:          42,
			UserID:      7,
			CategoryID:  1,
			Title:       "title",
			Description: "description",
			Status:      entity.IncidentStatusReview,
			AddressText: "Самара",
		},
	}
	uc := New(
		repo,
		userRepoStub{user: entity.User{ID: 7}},
		categoryRepoStub{},
		nil,
	)

	status := entity.IncidentStatusPublished
	_, err := uc.Update(context.Background(), 7, false, 42, entity.UpdateIncidentInput{Status: &status})

	require.ErrorIs(t, err, incidenterr.ErrForbidden)
	require.False(t, repo.updateCalled)
}

func TestRenderIncidentHTMLUsesImageTagWithoutDistortion(t *testing.T) {
	t.Parallel()

	html, err := renderIncidentHTML(buildIncidentDocumentView(entity.Incident{
		ID:             1,
		DepartmentName: "ГИБДД",
		CategoryTitle:  "Парковка",
		Title:          "Нарушение",
		Description:    "Описание",
		AddressText:    "Самара",
		ReporterEmail:  "user@example.com",
		Photos: []entity.IncidentPhoto{
			{FileURL: "https://example.com/photo.jpg"},
		},
	}))

	require.NoError(t, err)
	require.Contains(t, html, `<img src="https://example.com/photo.jpg" alt="Фотография инцидента">`)
	require.Contains(t, html, "object-fit: contain;")
	require.NotContains(t, html, "<a href=")
}

func TestNormalizePhotoContentType(t *testing.T) {
	t.Parallel()

	require.Equal(t, "image/png", normalizePhotoContentType("image/png"))
	require.Equal(t, "application/octet-stream", normalizePhotoContentType(""))
}

func TestEnsureCanViewAllowsPublishedForAnonymous(t *testing.T) {
	t.Parallel()

	err := ensureCanView(entity.Incident{
		Status: entity.IncidentStatusPublished,
	}, 0, false)

	require.NoError(t, err)
}

type incidentRepoStub struct {
	incident     entity.Incident
	updateCalled bool
}

func (s *incidentRepoStub) Create(_ context.Context, incident *entity.Incident) error {
	s.incident = *incident
	if s.incident.ID == 0 {
		s.incident.ID = 1
	}
	incident.ID = s.incident.ID
	return nil
}

func (s *incidentRepoStub) GetByID(_ context.Context, _ int64) (entity.Incident, error) {
	return s.incident, nil
}

func (s *incidentRepoStub) List(_ context.Context, _ entity.IncidentFilter) ([]entity.Incident, error) {
	return nil, nil
}

func (s *incidentRepoStub) Update(_ context.Context, incident *entity.Incident) error {
	s.updateCalled = true
	s.incident = *incident
	return nil
}

func (s *incidentRepoStub) Delete(_ context.Context, _ int64) error {
	return nil
}

func (s *incidentRepoStub) CreatePhoto(_ context.Context, _ *entity.IncidentPhoto) error {
	return nil
}

func (s *incidentRepoStub) DeletePhoto(_ context.Context, _, _ int64) (entity.IncidentPhoto, error) {
	return entity.IncidentPhoto{}, nil
}

type userRepoStub struct {
	user entity.User
}

func (s userRepoStub) Create(_ context.Context, _ *entity.User) error {
	return nil
}

func (s userRepoStub) Delete(_ context.Context, _ int64) error {
	return nil
}

func (s userRepoStub) GetByID(_ context.Context, _ int64) (entity.User, error) {
	return s.user, nil
}

func (s userRepoStub) GetByIdentifier(_ context.Context, identifier string) (entity.User, error) {
	if strings.TrimSpace(identifier) == "" {
		return entity.User{}, nil
	}
	return s.user, nil
}

func (s userRepoStub) CreateEmailVerificationCode(_ context.Context, _, _, _ string, _ int64) error {
	return nil
}

func (s userRepoStub) ConsumeEmailVerificationCode(_ context.Context, _, _, _ string, _ int64) error {
	return nil
}

func (s userRepoStub) List(_ context.Context) ([]entity.User, error) {
	return nil, nil
}

func (s userRepoStub) Update(_ context.Context, _ *entity.User) error {
	return nil
}

func (s userRepoStub) UpdateAvatar(_ context.Context, _ int64, _ string) error {
	return nil
}

type categoryRepoStub struct{}

func (categoryRepoStub) GetAll(_ context.Context) ([]entity.Category, error) {
	return nil, nil
}

func (categoryRepoStub) GetByID(_ context.Context, _ int) (entity.Category, error) {
	return entity.Category{ID: 1, Title: "Парковка"}, nil
}

func (categoryRepoStub) Create(_ context.Context, _ entity.CreateCategoryInput) (int, error) {
	return 0, nil
}

func (categoryRepoStub) Update(_ context.Context, _ int, _ entity.UpdateCategoryInput) error {
	return nil
}

func (categoryRepoStub) UpdateIcon(_ context.Context, _ int, _ *string) error {
	return nil
}

func (categoryRepoStub) Delete(_ context.Context, _ int) error {
	return nil
}
