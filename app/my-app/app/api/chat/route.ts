import { NextRequest, NextResponse } from 'next/server'
import { AIAgent } from '@/lib/ai-agent'

interface ChatMessage {
  id: string
  content: string
  sender: 'user' | 'bot'
  timestamp: Date
}

interface ChatRequest {
  message: string
  userId?: string
}

interface ChatResponse {
  response: string
  success: boolean
  error?: string
}

export async function POST(request: NextRequest) {
  try {
    const { message, userId }: ChatRequest = await request.json()

    if (!message || !message.trim()) {
      return NextResponse.json(
        { success: false, error: 'Message is required' },
        { status: 400 }
      )
    }

    // Get environment variables
    const openRouterApiKey = process.env.OPENROUTER_API_KEY
    const googleScriptURL = process.env.GOOGLE_SCRIPT_URL

    if (!openRouterApiKey) {
      console.error('OPENROUTER_API_KEY not configured')
      return NextResponse.json(
        { success: false, error: 'AI service not configured' },
        { status: 500 }
      )
    }

    if (!googleScriptURL) {
      console.error('GOOGLE_SCRIPT_URL not configured')
      return NextResponse.json(
        { success: false, error: 'Spreadsheet service not configured' },
        { status: 500 }
      )
    }

    // Create AI agent and process message
    const aiAgent = new AIAgent(openRouterApiKey, googleScriptURL)
    const response = await aiAgent.processMessage(message, userId)

    return NextResponse.json({
      success: true,
      response: response
    })

  } catch (error) {
    console.error('Chat API error:', error)
    return NextResponse.json(
      { 
        success: false, 
        error: 'Failed to process message. Please try again.' 
      },
      { status: 500 }
    )
  }
}

export async function GET() {
  return NextResponse.json({ 
    status: 'Chat API is running',
    timestamp: new Date().toISOString()
  })
} 