package hr

import "time"

type Resume struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Email     string   `json:"email"`
	Phone     string   `json:"phone"`
	Skills    []string `json:"skills"`
	Experience []string `json:"experience"`
	Education []string `json:"education"`
	RawText   string   `json:"raw_text"`
}

type Vacancy struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Requirements []string `json:"requirements"`
	Skills      []string `json:"skills"`
	Experience  string   `json:"experience"`
}

type Candidate struct {
	Resume   Resume   `json:"resume"`
	Vacancy  Vacancy  `json:"vacancy"`
	MatchScore float64 `json:"match_score"`
	Strengths []string `json:"strengths"`
	Weaknesses []string `json:"weaknesses"`
}

type Interview struct {
	ID          string    `json:"id"`
	CandidateID string    `json:"candidate_id"`
	VacancyID   string    `json:"vacancy_id"`
	CandidateName string  `json:"candidate_name"`
	Position    string    `json:"position"`
	ScheduledAt time.Time `json:"scheduled_at"`
	Duration    int       `json:"duration_minutes"`
	Location    string    `json:"location"`
	Interviewers []string `json:"interviewers"`
	Status      string    `json:"status"`
	Notes       string    `json:"notes"`
}

type Feedback struct {
	ID           string  `json:"id"`
	InterviewID string  `json:"interview_id"`
	CandidateID string  `json:"candidate_id"`
	Interviewer string  `json:"interviewer"`
	Rating      int     `json:"rating"`
	Pros        []string `json:"pros"`
	Cons        []string `json:"cons"`
	Recommendation string `json:"recommendation"`
	Comments    string  `json:"comments"`
	CreatedAt   time.Time `json:"created_at"`
}

type Task struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Payload   string `json:"payload"`
	TraceID   string `json:"trace_id"`
	SpanID    string `json:"span_id"`
}

type Result struct {
	TaskID  string `json:"task_id"`
	Success bool   `json:"success"`
	Output  string `json:"output"`
	Error   string `json:"error,omitempty"`
}

type PipelineTask struct {
	TaskID      string    `json:"task_id"`
	Resume      Resume    `json:"resume"`
	Vacancy     Vacancy   `json:"vacancy"`
	Candidate   *Candidate `json:"candidate,omitempty"`
	Interview   *Interview `json:"interview,omitempty"`
	Feedbacks   []Feedback `json:"feedbacks,omitempty"`
	FinalScore  float64   `json:"final_score"`
	Recommendation string `json:"recommendation"`
}

const (
	SubjectResumeParser   = "hr.resume.parse"
	SubjectMatcher        = "hr.vacancy.match"
	SubjectScheduler      = "hr.interview.schedule"
	SubjectFeedback       = "hr.feedback.collect"
	SubjectAuction        = "hr.auction.bid_request"
	SubjectAuctionResult  = "hr.auction.bid_result"
	SubjectTasksCompleted = "hr.tasks.completed"
	SubjectLogs           = "hr.logs.collect"
	SubjectEvents         = "hr.events.process"
	SubjectDetection      = "hr.attacks.detect"
	SubjectBlock          = "hr.traffic.block"
)
