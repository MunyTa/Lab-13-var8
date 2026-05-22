package hr

import (
	"fmt"
	"time"
)

type FeedbackScore struct {
	TechnicalSkills   int `json:"technical_skills"`
	Communication     int `json:"communication"`
	ProblemSolving    int `json:"problem_solving"`
	CultureFit        int `json:"culture_fit"`
	Overall           int `json:"overall"`
}

func CalculateAverageScore(scores []FeedbackScore) float64 {
	if len(scores) == 0 {
		return 0.0
	}

	total := 0
	for _, s := range scores {
		total += s.Overall
	}
	return float64(total) / float64(len(scores))
}

func GenerateRecommendation(avgScore float64) string {
	switch {
	case avgScore >= 9:
		return "Strong Hire"
	case avgScore >= 7:
		return "Hire"
	case avgScore >= 5:
		return "No Hire"
	default:
		return "Strong No Hire"
	}
}

func ProcessFeedback(feedbacks []Feedback) (float64, string) {
	if len(feedbacks) == 0 {
		return 0.0, "No feedback available"
	}

	scores := []FeedbackScore{}
	for _, fb := range feedbacks {
		scores = append(scores, FeedbackScore{
			Overall: fb.Rating,
		})
	}

	avgScore := CalculateAverageScore(scores)
	recommendation := GenerateRecommendation(avgScore)

	return avgScore, recommendation
}

func CreateSummaryReport(feedbacks []Feedback, avgScore float64, recommendation string) string {
	report := fmt.Sprintf("Interview Summary Report\n")
	report += fmt.Sprintf("Generated: %s\n\n", time.Now().Format(time.RFC822))

	report += fmt.Sprintf("Average Score: %.2f/10\n", avgScore)
	report += fmt.Sprintf("Recommendation: %s\n\n", recommendation)

	report += fmt.Sprintf("Individual Feedback (%d reviews):\n", len(feedbacks))
	for i, fb := range feedbacks {
		report += fmt.Sprintf("\n--- Review %d ---\n", i+1)
		report += fmt.Sprintf("Interviewer: %s\n", fb.Interviewer)
		report += fmt.Sprintf("Rating: %d/10\n", fb.Rating)
		if len(fb.Pros) > 0 {
			report += fmt.Sprintf("Pros: %s\n", joinStrings(fb.Pros, ", "))
		}
		if len(fb.Cons) > 0 {
			report += fmt.Sprintf("Cons: %s\n", joinStrings(fb.Cons, ", "))
		}
		if fb.Comments != "" {
			report += fmt.Sprintf("Comments: %s\n", fb.Comments)
		}
	}

	return report
}

func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}
