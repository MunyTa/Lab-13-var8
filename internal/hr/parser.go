package hr

import (
	"regexp"
	"strings"
)

type ParsedResume struct {
	Name       string
	Email      string
	Phone      string
	Skills     []string
	Experience []string
	Education  []string
}

func ParseResumeText(text string) *ParsedResume {
	pr := &ParsedResume{
		Skills:     []string{},
		Experience: []string{},
		Education:  []string{},
	}

	emailRegex := regexp.MustCompile(`[\w.-]+@[\w.-]+\.\w+`)
	if match := emailRegex.FindString(text); match != "" {
		pr.Email = strings.ToLower(match)
	}

	phoneRegex := regexp.MustCompile(`[\+]?[(]?[0-9]{1,3}[)]?[-\s\.]?[(]?[0-9]{1,3}[)]?[-\s\.]?[0-9]{3,6}[-\s\.]?[0-9]{3,6}`)
	pr.Phone = phoneRegex.FindString(text)

	lines := strings.Split(text, "\n")
	inExperience := false
	inEducation := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		lower := strings.ToLower(line)

		if strings.Contains(lower, "@") || strings.Contains(lower, "email") {
			continue
		}
		if phoneRegex.MatchString(line) {
			continue
		}

		if len(line) < 3 {
			continue
		}

		if strings.Contains(lower, "experience") || strings.Contains(lower, "работ") || strings.Contains(lower, "position") {
			inExperience = true
			inEducation = false
			continue
		}
		if strings.Contains(lower, "education") || strings.Contains(lower, "образовани") || strings.Contains(lower, "универ") {
			inEducation = true
			inExperience = false			continue
		}
		if strings.Contains(lower, "skill") || strings.Contains(lower, "компетенц") {
			inExperience = false
			inEducation = false
		}

		if inExperience {
			pr.Experience = append(pr.Experience, line)
		} else if inEducation {
			pr.Education = append(pr.Education, line)
		} else {
			pr.Skills = append(pr.Skills, line)
		}
	}

	skillKeywords := []string{"go", "golang", "python", "java", "javascript", "sql", "docker", "kubernetes",
		"git", "linux", "aws", "gcp", "redis", "postgresql", "mongodb", "microservices",
		"rest", "api", "ci/cd", "agile", "scrum", "team lead", "hr", "recruiting"}
	for _, skill := range skillKeywords {
		if strings.Contains(strings.ToLower(text), skill) {
			pr.Skills = append(pr.Skills, skill)
		}
	}

	return pr
}

func CalculateMatchScore(resumeSkills []string, vacancySkills []string) float64 {
	if len(vacancySkills) == 0 {
		return 0.0
	}

	matchCount := 0
	resumeLower := make([]string, len(resumeSkills))
	for i, s := range resumeSkills {
		resumeLower[i] = strings.ToLower(s)
	}

	for _, vacSkill := range vacancySkills {
		vacSkillLower := strings.ToLower(vacSkill)
		for _, resSkill := range resumeLower {
			if strings.Contains(resSkill, vacSkillLower) || strings.Contains(vacSkillLower, resSkill) {
				matchCount++
				break
			}
		}
	}

	return float64(matchCount) / float64(len(vacancySkills)) * 100.0
}
