# Google Apps Script Setup Guide

## Step 1: Create a New Google Spreadsheet

1. Go to [Google Sheets](https://sheets.google.com)
2. Create a new blank spreadsheet
3. Name it "TODO Bot Tasks" (or any name you prefer)
4. Note the spreadsheet ID from the URL (the long string between `/d/` and `/edit`)

## Step 2: Create the Apps Script

1. In your Google Spreadsheet, go to **Extensions** → **Apps Script**
2. Delete the default `myFunction()` code
3. Copy and paste the entire contents of `deploy_webhook.gs` into the script editor
4. Save the project (Ctrl+S or Cmd+S)
5. Name the project "TODO Bot Webhook"

## Step 3: Deploy as Web App

1. In the Apps Script editor, click **Deploy** → **New deployment**
2. Click the gear icon ⚙️ next to "Type" and select **Web app**
3. Configure the deployment:
   - **Description**: "TODO Bot Webhook v1"
   - **Execute as**: Me (your email)
   - **Who has access**: Anyone
4. Click **Deploy**
5. **IMPORTANT**: Copy the Web app URL that appears - this is your `GOOGLE_SCRIPT_URL`

## Step 4: Grant Permissions

1. You'll be prompted to authorize the script
2. Click **Review permissions**
3. Choose your Google account
4. Click **Advanced** → **Go to TODO Bot Webhook (unsafe)**
5. Click **Allow**

## Step 5: Update Your .env File

1. Open your `.env` file in the `go-todo-bot` directory
2. Replace the placeholder URL:
   ```
   GOOGLE_SCRIPT_URL=https://script.google.com/macros/s/YOUR_ACTUAL_SCRIPT_ID/exec
   ```
   With your actual Web app URL from Step 3

## Step 6: Test the Setup

Run this command to test your Google Apps Script:

```bash
# Test get_team action
curl -L -d '{"action":"get_team"}' YOUR_ACTUAL_SCRIPT_URL

# Expected response:
# {"success":true,"message":"Team retrieved","team":[...]}
```

## Troubleshooting

### Common Issues:

1. **"Script function not found"**
   - Make sure you copied the entire `deploy_webhook.gs` file
   - Ensure the function names match exactly

2. **"Permission denied"**
   - Redeploy the script with "Anyone" access
   - Make sure you authorized the script properly

3. **"Invalid JSON"**
   - Check that your curl command uses proper JSON format
   - Ensure you're using the correct Web app URL

4. **"Spreadsheet not found"**
   - Make sure the script is attached to the correct spreadsheet
   - The script will automatically create the necessary sheets

### Sheet Structure

The script will automatically create two sheets:

**"todo" sheet columns:**
- Timestamp
- People  
- Client
- Summary
- DueDate
- FullMessage
- Status (dropdown: Not Started/In Progress/Complete)
- BotNotes

**"team" sheet columns:**
- Name
- Email

### Sample Team Data

The script automatically adds sample team members:
- team, team@example.com
- lilly, lilly@example.com  
- jemma, jemma@example.com

You can edit these or add your own team members directly in the "team" sheet.

## Security Notes

- The Web app URL contains a secret token - keep it secure
- Only share the URL with trusted applications
- You can revoke access by creating a new deployment
- Monitor the Apps Script execution logs for any suspicious activity 