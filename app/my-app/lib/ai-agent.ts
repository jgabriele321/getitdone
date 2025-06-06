// AI Agent library - TypeScript port of the Go backend logic

export interface Task {
  people: string[]
  client: string
  summary: string
  dueDate: string
  confidence: number
}

export interface ParseResponse {
  tasks: Task[]
  originalMessage: string
}

export interface TaskRow {
  timestamp: string
  people: string[]
  client: string
  summary: string
  fullMessage: string
  status: string
  dueDate: string
  botNotes: string
}

export interface TeamMember {
  name: string
  email: string
}

export interface AddTasksResponse {
  status: string
  rowsAdded: number
  message?: string
  error?: string
}

// LLM Client - TypeScript port of internal/llm/client.go
export class LLMClient {
  private apiKey: string
  private baseURL: string = 'https://openrouter.ai/api/v1/chat/completions'
  private model: string = 'openai/gpt-4o-mini'

  constructor(apiKey: string) {
    this.apiKey = apiKey
  }

  async parseMessage(message: string): Promise<ParseResponse> {
    console.log('Parsing message with LLM:', message)

    const prompt = this.buildPrompt(message)

    const request = {
      model: this.model,
      messages: [
        {
          role: 'system',
          content: 'You are a task parser that ONLY returns valid JSON. Never include explanations or additional text.'
        },
        {
          role: 'user',
          content: prompt
        }
      ]
    }

    try {
      const response = await fetch(this.baseURL, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${this.apiKey}`
        },
        body: JSON.stringify(request)
      })

      if (!response.ok) {
        throw new Error(`API request failed with status ${response.status}`)
      }

      const data = await response.json()

      if (!data.choices || data.choices.length === 0) {
        throw new Error('No choices in response')
      }

      let content = data.choices[0].message.content

      // Clean the content - remove any non-JSON text
      const jsonMatch = content.match(/\{[\s\S]*\}/)
      if (jsonMatch) {
        content = jsonMatch[0]
      }

      // Try to parse the JSON response
      let parseResp: ParseResponse
      try {
        parseResp = JSON.parse(content)
      } catch (err) {
        console.warn('Failed to parse LLM response as JSON, creating fallback:', err)
        // Create fallback response
        parseResp = {
          tasks: [{
            people: ['team'],
            client: 'Internal',
            summary: this.extractSummary(message),
            dueDate: 'Unsure',
            confidence: 0.5
          }],
          originalMessage: message
        }
      }

      // Normalize and validate tasks
      parseResp.tasks = parseResp.tasks.map((task, i) => ({
        ...task,
        people: this.normalizeNames(task.people || ['team']),
        summary: this.truncateSummary(task.summary || 'Task'),
        client: task.client || (i > 0 && parseResp.tasks[0].client !== 'Internal' ? parseResp.tasks[0].client : 'Unsure'),
        dueDate: task.dueDate || 'Unsure',
        confidence: task.confidence || 0.8
      }))

      console.log('Successfully parsed message:', {
        taskCount: parseResp.tasks.length,
        tasks: parseResp.tasks
      })

      return parseResp
    } catch (error) {
      console.error('LLM parsing error:', error)
      throw error
    }
  }

  private buildPrompt(message: string): string {
    const currentTime = new Date().toISOString().split('T')[0]
    return `Parse this message into tasks and return ONLY a JSON object, no other text.

Current Date: ${currentTime}
Message: "${message}"

Rules:
1. Split multi-task messages into separate tasks (look for bullet points, "AND", or clear task boundaries)
2. For each task:
   - people: array of who is DOING the task (lowercase) or ["team"] if unclear
   - client: who the task is FOR. Important rules for client:
     * If message mentions specific client/company names, use that
     * For internal tasks, use "Internal"
     * If unclear, use "Unsure"
   - summary: clear, concise description of the task (max 100 chars)
   - dueDate: extract or infer due date. Format as "YYYY-MM-DD" or use "Unsure"
   - confidence: 0.0-1.0 based on how clear the task is

Return format:
{
  "tasks": [
    {
      "people": ["person1", "person2"],
      "client": "Client Name",
      "summary": "Task description",
      "dueDate": "2024-01-15",
      "confidence": 0.9
    }
  ],
  "originalMessage": "${message}"
}`
  }

  private normalizeNames(names: string[]): string[] {
    return names.map(name => name.toLowerCase().trim()).filter(name => name.length > 0)
  }

  private truncateSummary(summary: string): string {
    return summary.length > 100 ? summary.substring(0, 97) + '...' : summary
  }

  private extractSummary(message: string): string {
    // Simple extraction - take first 50 chars as summary
    return message.length > 50 ? message.substring(0, 47) + '...' : message
  }
}

// Google Sheets Client - TypeScript port of internal/sheets/client.go
export class SheetsClient {
  private webhookURL: string

  constructor(webhookURL: string) {
    this.webhookURL = webhookURL
  }

  async addTasks(tasks: TaskRow[]): Promise<void> {
    console.log('Adding tasks to Google Sheets:', tasks.length)

    const request = {
      action: 'add_tasks',
      tasks: tasks
    }

    try {
      const response = await fetch(this.webhookURL, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify(request)
      })

      if (!response.ok) {
        throw new Error(`HTTP request failed with status ${response.status}`)
      }

      const data: AddTasksResponse = await response.json()

      if (data.status !== 'success') {
        throw new Error(`Sheets API error: ${data.error}`)
      }

      console.log('Successfully added tasks to Google Sheets:', {
        rowsAdded: data.rowsAdded
      })
    } catch (error) {
      console.error('Sheets client error:', error)
      throw error
    }
  }

  async getTeam(): Promise<TeamMember[]> {
    console.log('Getting team data from Google Sheets')

    const request = {
      action: 'get_team'
    }

    try {
      const response = await fetch(this.webhookURL, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify(request)
      })

      if (!response.ok) {
        throw new Error(`HTTP request failed with status ${response.status}`)
      }

      const data = await response.json()

      if (data.status !== 'success') {
        throw new Error(`Sheets API error: ${data.error}`)
      }

      console.log('Successfully retrieved team data:', {
        teamSize: data.team.length
      })

      return data.team
    } catch (error) {
      console.error('Team retrieval error:', error)
      throw error
    }
  }
}

// Utility function to create task rows
export function createTaskRow(
  people: string[],
  client: string,
  summary: string,
  fullMessage: string,
  dueDate: string,
  botNotes: string
): TaskRow {
  return {
    timestamp: new Date().toISOString(),
    people,
    client,
    summary,
    fullMessage,
    status: 'Not Started',
    dueDate,
    botNotes
  }
}

// Main AI Agent class that orchestrates everything
export class AIAgent {
  private llmClient: LLMClient
  private sheetsClient: SheetsClient

  constructor(openRouterApiKey: string, googleScriptURL: string) {
    this.llmClient = new LLMClient(openRouterApiKey)
    this.sheetsClient = new SheetsClient(googleScriptURL)
  }

  async processMessage(message: string, userId?: string): Promise<string> {
    try {
      console.log('Processing message:', { message, userId })

      // Parse message with LLM
      const parseResp = await this.llmClient.parseMessage(message)

      // Convert to sheet rows
      const taskRows: TaskRow[] = []
      for (let i = 0; i < parseResp.tasks.length; i++) {
        const parsedTask = parseResp.tasks[i]
        let summary = parsedTask.summary
        
        if (parseResp.tasks.length > 1) {
          summary = `${parsedTask.summary} (${i + 1}/${parseResp.tasks.length})`
        }

        let botNotes = `Web User: ${userId || 'anonymous'}, Confidence: ${parsedTask.confidence.toFixed(2)}`
        if (parsedTask.confidence < 0.7) {
          botNotes += ' (Low confidence)'
        }

        const taskRow = createTaskRow(
          parsedTask.people,
          parsedTask.client,
          summary,
          message,
          parsedTask.dueDate,
          botNotes
        )
        taskRows.push(taskRow)
      }

      // Save to Google Sheets
      await this.sheetsClient.addTasks(taskRows)

      // Generate response message
      const taskCount = parseResp.tasks.length
      if (taskCount === 1) {
        return `✅ I've processed your task and added it to your productivity tracker. The task has been assigned to ${parseResp.tasks[0].people.join(', ')} for ${parseResp.tasks[0].client}.`
      } else {
        return `✅ I've processed your message and created ${taskCount} tasks in your productivity tracker. Each task has been properly categorized and assigned. Check your spreadsheet for the complete breakdown.`
      }

    } catch (error) {
      console.error('AI Agent processing error:', error)
      throw new Error('Failed to process your message. Please try again.')
    }
  }
} 