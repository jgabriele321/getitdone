"use client"

import type React from "react"
import Image from "next/image"
import { useState } from "react"
import { Button } from "../components/ui/button"
import { Textarea } from "../components/ui/textarea"
import { Menu, Send } from "lucide-react"
import { Sheet, SheetContent, SheetTrigger } from "../components/ui/sheet"

interface Message {
  id: string
  content: string
  sender: "user" | "bot"
  timestamp: Date
}

export default function ChatPage() {
  const [messages, setMessages] = useState<Message[]>([])
  const [inputValue, setInputValue] = useState("")
  const [isLoading, setIsLoading] = useState(false)

  const handleSubmit = async () => {
    if (!inputValue.trim()) return

    const userMessage: Message = {
      id: Date.now().toString(),
      content: inputValue.trim(),
      sender: "user",
      timestamp: new Date(),
    }

    setMessages((prev) => [...prev, userMessage])
    const messageContent = inputValue.trim()
    setInputValue("")
    setIsLoading(true)

    try {
      const response = await fetch('/api/chat', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          message: messageContent,
          userId: 'web-user' // Optional: can be used for user tracking
        }),
      })

      const data = await response.json()

      if (data.success) {
        const botMessage: Message = {
          id: (Date.now() + 1).toString(),
          content: data.response,
          sender: "bot",
          timestamp: new Date(),
        }
        setMessages((prev) => [...prev, botMessage])
      } else {
        // Handle API error
        const errorMessage: Message = {
          id: (Date.now() + 1).toString(),
          content: data.error || "Sorry, I encountered an error processing your request. Please try again.",
          sender: "bot",
          timestamp: new Date(),
        }
        setMessages((prev) => [...prev, errorMessage])
      }
    } catch (error) {
      console.error('Error calling chat API:', error)
      const errorMessage: Message = {
        id: (Date.now() + 1).toString(),
        content: "Sorry, I'm having trouble connecting right now. Please try again.",
        sender: "bot",
        timestamp: new Date(),
      }
      setMessages((prev) => [...prev, errorMessage])
    } finally {
      setIsLoading(false)
    }
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault()
      handleSubmit()
    }
  }

  return (
    <div className="min-h-screen bg-white text-black">
      {/* Header */}
      <header className="flex items-center justify-between p-3 md:p-5 border-b border-gray-100">
        <div className="flex items-center gap-3">
          <Image src="/logo-black.svg" alt="Shift6 Studios" width={36} height={36} className="h-7 md:h-9" />
          <h1 className="text-lg md:text-2xl font-bold tracking-tight">Productivity Hub</h1>
        </div>

        <Sheet>
          <SheetTrigger asChild>
            <Button variant="ghost" size="icon" className="hover:bg-gray-100">
              <Menu className="h-6 w-6" />
              <span className="sr-only">Open menu</span>
            </Button>
          </SheetTrigger>
          <SheetContent side="right" className="w-64 bg-white">
            <nav className="mt-8">
              <Button
                variant="ghost"
                className="w-full justify-start text-lg font-medium hover:bg-gray-100"
                onClick={() => window.location.reload()}
              >
                Home
              </Button>
            </nav>
          </SheetContent>
        </Sheet>
      </header>

      <main className="container mx-auto px-4 py-6 md:py-9 max-w-4xl">
        {/* Chat Section */}
        <div className="space-y-6 md:space-y-7">
          <div className="text-center space-y-3">
            <h2 className="text-3xl md:text-4xl font-black tracking-tight">Task Manager</h2>
            <a
              href="https://docs.google.com/spreadsheets/d/1xq52BtoOmUIFiZ3BR_v5_Bay7jQSi5qZsT9a0pLvKBA/edit?gid=94316419#gid=94316419"
              target="_blank"
              rel="noopener noreferrer"
              className="text-lg md:text-xl text-black hover:text-gray-600 underline transition-colors"
            >
              Spreadsheet Here
            </a>
          </div>

          {/* Chat Messages */}
          <div className="bg-gray-50 rounded-lg p-5 h-[270px] md:h-[360px] overflow-y-auto">
            <div className="space-y-3">
              {messages.length === 0 ? (
                <div className="flex items-center justify-center h-full text-gray-400">
                  <p className="text-center">Start by adding your to-do items below.</p>
                </div>
              ) : (
                messages.map((message) => (
                  <div key={message.id} className={`flex ${message.sender === "user" ? "justify-end" : "justify-start"}`}>
                    <div
                      className={`max-w-[80%] p-3 rounded-lg ${
                        message.sender === "user" ? "bg-black text-white" : "bg-white border border-gray-200"
                      }`}
                    >
                      <p className="whitespace-pre-wrap">{message.content}</p>
                      <span className="text-xs opacity-70 mt-2 block">{message.timestamp.toLocaleTimeString()}</span>
                    </div>
                  </div>
                ))
                            )}
              {isLoading && (
                <div className="flex justify-start">
                  <div className="bg-white border border-gray-200 p-4 rounded-lg">
                    <div className="flex space-x-1">
                      <div className="w-2 h-2 bg-gray-400 rounded-full animate-bounce"></div>
                      <div
                        className="w-2 h-2 bg-gray-400 rounded-full animate-bounce"
                        style={{ animationDelay: "0.1s" }}
                      ></div>
                      <div
                        className="w-2 h-2 bg-gray-400 rounded-full animate-bounce"
                        style={{ animationDelay: "0.2s" }}
                      ></div>
                    </div>
                  </div>
                </div>
              )}
            </div>
          </div>

          {/* Input Section */}
          <div className="space-y-3">
            <Textarea
              value={inputValue}
              onChange={(e) => setInputValue(e.target.value)}
              onKeyDown={handleKeyDown}
              placeholder="Add to do list items here, one at a time or separated by bullet points or dashes"
              className="min-h-[110px] text-base resize-none border-2 border-gray-200 focus:border-black transition-colors"
              disabled={isLoading}
            />
            <div className="flex justify-end">
              <Button
                onClick={handleSubmit}
                disabled={!inputValue.trim() || isLoading}
                className="bg-black hover:bg-gray-800 text-white px-7 py-2 text-base font-medium"
              >
                <Send className="w-4 h-4 mr-2" />
                Submit
              </Button>
            </div>
          </div>
        </div>
      </main>
    </div>
  )
}
