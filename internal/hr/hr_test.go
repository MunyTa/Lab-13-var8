package hr

import (
	"testing"
)

func TestParseResumeText(t *testing.T) {
	text := `
John Doe
Email: john.doe@example.com
Phone: +1-555-123-4567

Skills:
- Go programming
- Python development
- Docker and Kubernetes
- SQL databases

Experience:
- Senior Developer at Tech Corp (2020-2024)
- Software Engineer at Startup Inc (2018-2020)

Education:
- BSc Computer Science, MIT (2014-2018)
`

	pr := ParseResumeText(text)

	if pr.Email != "john.doe@example.com" {
		t.Errorf("Expected email john.doe@example.com, got %s", pr.Email)
	}

	if pr.Phone == "" {
		t.Error("Phone number should be extracted")
	}

	if len(pr.Skills) == 0 {
		t.Error("Skills should be parsed")
	}
}

func TestCalculateMatchScore(t *testing.T) {
	resumeSkills := []string{"Go", "Python", "Docker", "SQL"}
	vacancySkills := []string{"Go", "Python", "Kubernetes", "AWS"}

	score := CalculateMatchScore(resumeSkills, vacancySkills)

	expected := 50.0
	if score != expected {
		t.Errorf("Expected score %f, got %f", expected, score)
	}
}

func TestMatchCandidateToVacancy(t *testing.T) {
	resume := Resume{
		ID:     "res-001",
		Skills: []string{"Go", "Python", "Docker"},
	}
	vacancy := Vacancy{
		ID:     "vac-001",
		Title:  "Backend Developer",
		Skills: []string{"Go", "Docker", "Kubernetes"},
	}

	result := MatchCandidateToVacancy(resume, vacancy)

	if result.CandidateID != resume.ID {
		t.Errorf("Expected candidate ID %s, got %s", resume.ID, result.CandidateID)
	}

	if len(result.Strengths) != 2 {
		t.Errorf("Expected 2 strengths, got %d", len(result.Strengths))
	}
}

func TestFindBestCandidates(t *testing.T) {
	results := []MatchResult{
		{CandidateID: "c1", MatchScore: 75.0},
		{CandidateID: "c2", MatchScore: 90.0},
		{CandidateID: "c3", MatchScore: 60.0},
		{CandidateID: "c4", MatchScore: 85.0},
	}

	best := FindBestCandidates(results, 2)

	if len(best) != 2 {
		t.Errorf("Expected 2 best candidates, got %d", len(best))
	}

	if best[0].CandidateID != "c2" {
		t.Errorf("Expected c2 to be first, got %s", best[0].CandidateID)
	}

	if best[1].CandidateID != "c4" {
		t.Errorf("Expected c4 to be second, got %s", best[1].CandidateID)
	}
}

func TestGenerateTimeSlots(t *testing.T) {
	date := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	slots := GenerateTimeSlots(date, 9, 17, 60)

	expectedSlots := 8
	if len(slots) != expectedSlots {
		t.Errorf("Expected %d slots, got %d", expectedSlots, len(slots))
	}
}

func TestProcessFeedback(t *testing.T) {
	feedbacks := []Feedback{
		{Rating: 8, Pros: []string{"Good"}, Cons: []string{"Slow"}},
		{Rating: 9, Pros: []string{"Excellent"}, Cons: []string{"None"}},
	}

	avgScore, recommendation := ProcessFeedback(feedbacks)

	expectedScore := 8.5
	if avgScore != expectedScore {
		t.Errorf("Expected average score %f, got %f", expectedScore, avgScore)
	}

	if recommendation != "Hire" {
		t.Errorf("Expected 'Hire', got '%s'", recommendation)
	}
}
