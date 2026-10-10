package api

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rhl/businessos-backend/internal/founderos/pages/skills"
)

// /os/skills (spec 6.19): FounderOS v1 app/skills/page.tsx + /api/skills/[slug].
// The Claude Code skills are read from this host's ~/.claude (user + plugin
// skills), the operator skills from founderos_skills in FounderOS.
func init() { RegisterPage(registerSkills) }

// skillsCard is FounderOS v1 SkillsGrid's SkillCard.
type skillsCard struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Group       string  `json:"group"`
	Kind        string  `json:"kind"` // claude | operator
	Description string  `json:"description"`
	Meta        string  `json:"meta"`
	FilePath    string  `json:"filePath"`
	Status      *string `json:"status,omitempty"`
	Markdown    *string `json:"markdown,omitempty"`
}

var skillsTrailingWord = regexp.MustCompile(`\s+\S*$`)

// skillsTruncate cuts at n runes, back to a word boundary, with an ellipsis.
func skillsTruncate(t string, n int) string {
	r := []rune(t)
	if len(r) <= n {
		return t
	}
	return skillsTrailingWord.ReplaceAllString(string(r[:n]), "") + "…"
}

// skillDirs reads FounderOS v1's skills/plugins dir overrides through the
// bridge's Resolver (planted keys, env.local, then the process env).
func skillDirs(d *Deps) skills.Dirs {
	return skills.DirsFrom(func(k string) string { return d.Resolver.Resolve(k) })
}

func registerSkills(s *gin.RouterGroup, d *Deps) {
	s.GET("/pages/skills", func(c *gin.Context) {
		dirs := skillDirs(d)
		real := append(skills.ReadUserSkills(dirs), skills.ReadPluginSkills(dirs)...)
		cards := []skillsCard{}
		for _, sk := range real {
			cards = append(cards, skillsCard{ID: sk.Slug, Name: sk.Name, Group: sk.Group, Kind: "claude",
				Description: skillsTruncate(sk.Description, 110), Meta: sk.Path, FilePath: sk.Path})
		}

		var operator []skills.OperatorSkill
		names := map[string]string{}
		var opErr *string
		if d.Pool == nil {
			msg := "operator skills unavailable: no Postgres"
			opErr = &msg
		} else if rows, n, err := skills.LoadOperator(c.Request.Context(), d.Pool, "founderos"); err != nil {
			msg := "operator skills unavailable: " + err.Error()
			opErr = &msg
		} else {
			operator, names = rows, n
		}
		for _, sk := range operator {
			meta := "unassigned"
			if sk.OwnerAgentID != nil {
				meta = *sk.OwnerAgentID
				if name, ok := names[*sk.OwnerAgentID]; ok {
					meta = name
				}
			}
			status, md := sk.Status, sk.Markdown
			cards = append(cards, skillsCard{ID: sk.ID, Name: sk.Name, Group: "Operator · " + sk.Category, Kind: "operator",
				Description: skillsTruncate(sk.Description, 110), Meta: meta, FilePath: "skills/" + sk.ID + "/SKILL.md",
				Status: &status, Markdown: &md})
		}

		opNote := fmt.Sprintf("%d operator skills", len(operator))
		if opErr != nil {
			opNote = "operator skills unavailable"
		}
		note := fmt.Sprintf("%s (no ~/.claude/skills on this machine) · open any card to read or download its SKILL.md.", opNote)
		if len(real) > 0 {
			note = fmt.Sprintf("%d skills live from ~/.claude (user + plugins) + %s · open any card to read or download its SKILL.md.", len(real), opNote)
		}
		c.JSON(http.StatusOK, gin.H{
			"cards":         cards,
			"sourceNote":    note,
			"operatorError": opErr,
			"volume":        skills.Volume(skills.VolumeInput{Claude: real, Operator: operator, OperatorKnown: opErr == nil, AgentNames: names}),
		})
	})

	// The full SKILL.md: JSON for the reader, ?download=1 for the raw file.
	s.GET("/pages/skills/:slug", func(c *gin.Context) {
		slug := c.Param("slug")
		md, ok := skills.ReadMarkdown(skillDirs(d), slug)
		if !ok {
			c.JSON(http.StatusNotFound, gin.H{"error": "skill not found"})
			return
		}
		if c.Query("download") != "" {
			c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s-SKILL.md"`, strings.ReplaceAll(slug, ":", "-")))
			c.Data(http.StatusOK, "text/markdown; charset=utf-8", []byte(md))
			return
		}
		c.JSON(http.StatusOK, gin.H{"markdown": md})
	})
}
