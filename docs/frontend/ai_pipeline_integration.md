# Frontend Integration Guide: AI Pipeline Configuration

**Version**: 1.4.0  
**Date**: 2026-01-02

---

## Chat Input Changes

### New Prompt Prefixes

Users can now use special prefixes to control AI behavior:

```typescript
// Example usage
const examples = [
  "/bypass What is 2+2?",           // Pure LLM mode
  "/nuance:engineering Analyze X",  // Engineering tone
  "/nuance:creative Write a poem",  // Creative mode
  "What's in my notes?"             // Default RAG mode
];
```

### UI Considerations

1. **Fetch nuances from API** - Use `GET /api/chatbot/v1/nuances` to get available options
2. **Autocomplete** (optional): Show prefix suggestions when user types `/`
3. **Mode indicator**: Display current mode in chat bubble
4. **Help tooltip**: Explain prefix options to users

### Fetching Available Nuances

```typescript
// Fetch from API - NOT hardcoded!
const response = await fetch('/api/chatbot/v1/nuances', {
  headers: { 'Authorization': `Bearer ${token}` }
});
const { data } = await response.json();
// data: [{ key: "engineering", name: "Engineering Mode", description: "..." }, ...]
```

---

## Response Type Updates

### Updated `SendChatResponse`

```typescript
interface SendChatResponse {
  reply: string;
  citations?: CitationDTO[];
  mode?: "rag" | "bypass" | "nuance";  // NEW
  nuance_key?: string;                  // NEW
}
```

### Display Logic

```typescript
function getChatModeLabel(response: SendChatResponse): string {
  switch (response.mode) {
    case "bypass":
      return "💬 Direct Chat";
    case "nuance":
      return `🎭 ${response.nuance_key} mode`;
    default:
      return "📚 Notes";
  }
}
```

---

## Admin Panel: AI Configuration

### New Routes (Admin Only)

| Route | Component |
|-------|-----------|
| `/admin/ai/configurations` | AI Settings Table |
| `/admin/ai/nuances` | Nuance Management |

### API Endpoints

```typescript
// GET /api/admin/ai/configurations
interface AiConfigurationResponse {
  id: string;
  key: string;
  value: string;
  value_type: "string" | "number" | "boolean" | "json";
  description: string;
  category: "rag" | "llm" | "bypass" | "nuance";
  updated_at: string;
}

// PUT /api/admin/ai/configurations/:key
interface UpdateAiConfigurationRequest {
  value: string;
}

// GET /api/admin/ai/nuances
interface AiNuanceResponse {
  id: string;
  key: string;
  name: string;
  description: string;
  system_prompt: string;
  model_override?: string;
  is_active: boolean;
  sort_order: number;
  created_at: string;
  updated_at: string;
}

// POST /api/admin/ai/nuances
interface CreateAiNuanceRequest {
  key: string;         // e.g., "engineering"
  name: string;        // e.g., "Engineering Mode"
  description?: string;
  system_prompt: string;
  model_override?: string;
  sort_order?: number;
}

// PUT /api/admin/ai/nuances/:id
interface UpdateAiNuanceRequest {
  name?: string;
  description?: string;
  system_prompt?: string;
  model_override?: string;
  is_active?: boolean;
  sort_order?: number;
}
```

---

## Suggested UI Components

### 1. Chat Mode Badge

```tsx
// ChatBubble.tsx
function ChatModeBadge({ mode, nuanceKey }: { mode?: string; nuanceKey?: string }) {
  if (mode === "bypass") {
    return <Badge variant="secondary">💬 Direct</Badge>;
  }
  if (mode === "nuance") {
    return <Badge variant="accent">🎭 {nuanceKey}</Badge>;
  }
  return null; // RAG mode doesn't need badge
}
```

### 2. Prefix Helper

```tsx
// ChatInput.tsx
function PrefixHelper() {
  return (
    <Popover>
      <PopoverTrigger>
        <Button variant="ghost" size="icon">
          <SlashIcon />
        </Button>
      </PopoverTrigger>
      <PopoverContent>
        <ul>
          <li><code>/bypass</code> - Chat without notes</li>
          <li><code>/nuance:engineering</code> - Technical mode</li>
          <li><code>/nuance:creative</code> - Creative mode</li>
          <li><code>/nuance:formal</code> - Professional tone</li>
        </ul>
      </PopoverContent>
    </Popover>
  );
}
```

### 3. Admin Nuance Editor

```tsx
// AdminNuanceForm.tsx
function NuanceForm({ nuance }: { nuance?: AiNuanceResponse }) {
  return (
    <Form>
      <FormField name="key" label="Key" placeholder="e.g., engineering" />
      <FormField name="name" label="Display Name" />
      <FormTextarea 
        name="system_prompt" 
        label="System Prompt"
        rows={8}
        placeholder="You are a senior software engineer..."
      />
      <FormSwitch name="is_active" label="Active" />
    </Form>
  );
}
```

---

## Available Nuances

Nuances are server-defined behaviors that can:
1. Inject specific System Prompts (e.g., "You are a senior engineer")
2. **Switch underlying AI Models** (e.g., use `deepseek-coder` for engineering queries)

| Key | Purpose | Example Prompt |
|-----|---------|----------------|
| `engineering` | Technical analysis | `/nuance:engineering Review this code` |
| `creative` | Brainstorming | `/nuance:creative Generate ideas` |
| `formal` | Professional tone | `/nuance:formal Write an email` |
| `academic` | Scholarly style | `/nuance:academic Explain this concept` |
| `concise` | Brief responses | `/nuance:concise Summarize this` |

---
The AI model switch is configured by the ADMIN, but triggered by the USER choosing a nuance.

How It Works
Admin Side (Configuration): The Admin decides which model is best for a specific task.
Admin configures: "The 'engineering' nuance should use the deepseek-coder model."
Admin configures: "The 'creative' nuance should use llama3."
Admin configures: "The 'standard' nuance uses the default qwen2.5."
User Side (Chat): The User simply chooses the nuance (style/mode). They do not choose the model directly.
User types: /nuance:engineering How do I fix this bug?
System Action: "Oh, the user wants 'engineering' mode. The Admin set this to use deepseek-coder. I will switch to that model for this request."
Summary
User chooses the Goal/Style (e.g., "Engineering", "Creative").
Admin chooses the Model that powers that goal (e.g., "DeepSeek", "Llama3").
This separates the technical detail (model selection) from the user experience (picking a helpful mode). The user doesn't need to know what "DeepSeek" is, they just know "Engineering mode is good for code."

## Migration Notes

1. **No breaking changes** - Existing functionality preserved
2. **New optional fields** - `mode` and `nuance_key` in responses
3. **Admin routes** - Add to admin panel navigation
