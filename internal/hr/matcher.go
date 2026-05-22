package hr

import (
	"sort"
	"strings"
)

type MatchResult struct {
	CandidateID   string    `json:"candidate_id"`
	Resume        Resume    `json:"resume"`
	VacancyID     string    `json:"vacancy_id"`
	MatchScore    float64   `json:"match_score"`
	Strengths     []string  `json:"strengths"`
	Weaknesses    []string  `json:"weaknesses"`
}

func MatchCandidateToVacancy(resume Resume, vacancy Vacancy) MatchResult {
	result := MatchResult{
		CandidateID: resume.ID,
		Resume:      resume,
		VacancyID:   vacancy.ID,
		Strengths:   []string{},
		Weaknesses:  []string{},
	}

	resumeSkillsSet := make(map[string]bool)
	for _, skill := range resume.Skills {
		resumeSkillsSet[strings.ToLower(skill)] = true
	}

	vacancySkillsSet := make(map[string]bool)
	for _, skill := range vacancy.Skills {
		vacancySkillsSet[strings.ToLower(skill)] = true
	}

	for _, skill := range vacancy.Skills {
		if resumeSkillsSet[strings.ToLower(skill)] {
			result.Strengths = append(result.Strengths, skill)
		}
	}

	for _, skill := range vacancy.Skills {
		if !resumeSkillsSet[strings.ToLower(skill)] {
			result.Weaknesses = append(result.Weaknesses, skill)
		}
	}

	if len(vacancy.Skills) > 0 {
		result.MatchScore = float64(len(result.Strengths)) / float64(len(vacancy.Skills)) * 100.0
	} else {
		result.MatchScore = 50.0
	}

	sort.Strings(result.Strengths)
	sort.Strings(result.Weaknesses)

	return result
}

func FindBestCandidates(results []MatchResult, topN int) []MatchResult {
	sort.Slice(results, func(i, j int) bool {
		return results[i].MatchScore > results[j].MatchScore
	})

	if len(results) > topN {
		return results[:topN]
	}
	return results
}
