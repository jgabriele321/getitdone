# Google Sheets & Apps Script Setup Guide

This guide will help you set up the Google Sheets backend for the Go TODO Bot.

## Step 1: Create Google Sheet

1. Go to [Google Sheets](https://sheets.google.com)
2. Create a new blank spreadsheet
3. Name it "TODO Bot Tasks" (or your preferred name)
4. Create two tabs at the bottom:
   - Rename "Sheet1" to **"todo"**
   - Add a new sheet and name it **"team"**

## Step 2: Set Up Apps Script

1. In your Google Sheet, go to **Extensions → Apps Script**
2. Delete any existing code in the editor
3. Copy the entire contents of `deploy_webhook.gs`
4. Paste it into the Apps Script editor
5. Save the project (Ctrl+S or Cmd+S)
6. Name the project "TODO Bot Webhook"

## Step 3: Initialize the Sheets

1. In the Apps Script editor, find the function dropdown (near the top)
2. Select `initializeSheets` from the dropdown
3. Click the **Run** button
4. You'll be prompted to authorize the script:
   - Review permissions
   - Choose your Google account
   - Click "Advanced" → "Go to TODO Bot Webhook (unsafe)"
   - Click "Allow"
5. Check your Google Sheet - you should now see:
   - Headers in both sheets
   - Sample team members in the team sheet
   - Data validation in the Status column

## Step 4: Deploy as Web App

1. In the Apps Script editor, click **Deploy → New Deployment**
2. Click the gear icon next to "Select type" and choose **Web app**
3. Configure the deployment:
   - Description: "TODO Bot Webhook v1"
   - Execute as: **Me** (your email)
   - Who has access: **Anyone**
4. Click **Deploy**
5. **IMPORTANT**: Copy the Web app URL (looks like `https://script.google.com/macros/s/XXXXX/exec`)
6. Save this URL - you'll need it for the `GOOGLE_SCRIPT_URL` environment variable

## Step 5: Add Your Team Members

1. Go back to your Google Sheet
2. Click on the **team** tab
3. Replace the sample data with your actual team:
   - Column A: Name (lowercase, single word)
   - Column B: Email address
   - Example:
     ```
     alice    alice@company.com
     bob      bob@company.com
     sarah    sarah@company.com
     team     (leave email blank for team)
     ```

## Step 6: Test the Webhook

1. Back in Apps Script, select `testScript` from the function dropdown
2. Click **Run**
3. Check the execution log (View → Execution log)
4. Go to your Google Sheet's "todo" tab
5. You should see a test task added!

## Step 7: Configure the Bot

1. Add the Web app URL to your `.env` file:
   ```
   GOOGLE_SCRIPT_URL=https://script.google.com/macros/s/YOUR_SCRIPT_ID/exec
   ```

## Troubleshooting

### "Missing required sheets" error
- Make sure you have both "todo" and "team" tabs (case-sensitive)

### Authorization issues
- Ensure you're logged into the correct Google account
- The Web app must be set to "Anyone" for access

### No data appearing
- Check the execution log in Apps Script for errors
- Verify the JSON payload structure matches the expected format

### Making changes to the script
- After any code changes, you must create a **New Deployment**
- The deployment URL will remain the same if you update an existing deployment

## Security Notes

- The "Anyone" access setting means anyone with the URL can POST data
- Consider implementing a simple API key check if needed
- Keep your Web app URL private
- For production use, consider using a service account instead

## Next Steps

Once set up, your bot will be able to:
- Add tasks to the todo sheet
- Fetch team members for email distribution
- Query tasks by status for daily digests

The Google Sheet serves as both a database and a UI for manual task management! 