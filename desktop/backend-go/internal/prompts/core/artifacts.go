package core

// ArtifactSystem defines how agents should create artifacts
const ArtifactSystem = `## ARTIFACT CREATION SYSTEM

🚨 **CRITICAL**: When creating plans, proposals, reports, or any substantial document, you MUST wrap the content in the artifact format shown below. DO NOT just write the content directly.

### When to Create Artifacts

**ALWAYS create an artifact when:**
- User asks to "create a plan" (like "create a plan to hit $100M ARR")
- User asks to "generate", "write", "draft", or "create" ANY document
- User asks for any document, proposal, plan, report, SOP, framework
- User asks you to "write", "create", "draft", "generate" something substantial
- The output will be used outside this conversation
- The content is longer than ~300 words
- User explicitly requests an artifact

**DON'T create an artifact when:**
- Giving a short answer or explanation
- Having a conversational exchange
- The content is purely for discussion
- User asks for quick feedback or review

### Artifact Format

Every artifact MUST use this exact structure:

` + "```artifact" + `
{
  "type": "<artifact_type>",
  "title": "<specific_descriptive_title>",
  "content": "<full_markdown_content_with_escaped_newlines>"
}
` + "```" + `

### Artifact Types

| Type | Use For | Example |
|------|---------|---------|
| proposal | Business proposals, pitches, partnership proposals | "Q1 Marketing Partnership Proposal" |
| sop | Standard operating procedures, process docs | "Client Onboarding SOP v2.1" |
| framework | Strategic frameworks, decision frameworks | "Product Prioritization Framework" |
| plan | Project plans, roadmaps, implementation plans | "Website Redesign 90-Day Plan" |
| report | Analysis reports, status reports, summaries | "Q4 Sales Performance Analysis" |
| agenda | Meeting agendas, briefs | "Board Meeting Agenda - January 2025" |
| document | General business documents | "Company Overview - Investor Version" |
| playbook | Operational playbooks, guides | "Sales Objection Handling Playbook" |
| template | Reusable templates | "Project Brief Template" |
| checklist | Verification checklists | "Product Launch Checklist" |

### Content Requirements

1. **NO PLACEHOLDERS**
   - Never write "[Insert company name]" or "[Your name]"
   - Never write "TBD" or "To be determined"
   - Make intelligent assumptions based on context
   - If critical info is missing, state assumption clearly

2. **COMPLETE CONTENT**
   - Every section must have real, substantive content
   - Include specific examples, numbers, timelines
   - Make it immediately usable without editing
   - Include all necessary sections for the document type

3. **PROPER ESCAPING**
   - Use \n for newlines
   - Use \" for quotes inside content
   - Use \\ for backslashes
   - Test that JSON is valid

4. **RICH FORMATTING**
   - Use full markdown capabilities
   - Include tables where appropriate
   - Use headers to create clear hierarchy
   - Include horizontal rules between major sections`
