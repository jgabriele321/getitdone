/**
 * Simple Google Apps Script Webhook for Go TODO Bot
 */

function doPost(e) {
  try {
    // Get the request data
    let data;
    if (e.postData && e.postData.contents) {
      data = JSON.parse(e.postData.contents);
    } else {
      throw new Error("No POST data received");
    }
    
    // Open the spreadsheet
    const sheet = SpreadsheetApp.getActiveSpreadsheet();
    const todoSheet = sheet.getSheetByName('todo');
    const teamSheet = sheet.getSheetByName('team');
    
    // Check sheets exist
    if (!todoSheet || !teamSheet) {
      throw new Error('Missing required sheets. Please create "todo" and "team" tabs.');
    }
    
    // Handle different actions
    if (data.action === 'add_tasks') {
      return handleAddTasks(todoSheet, data.tasks);
    } else if (data.action === 'get_team') {
      return handleGetTeam(teamSheet);
    } else if (data.action === 'get_tasks') {
      return handleGetTasks(todoSheet, data.status);
    } else {
      throw new Error('Unknown action: ' + data.action);
    }
    
  } catch (error) {
    return ContentService
      .createTextOutput(JSON.stringify({
        status: 'error',
        message: error.message
      }))
      .setMimeType(ContentService.MimeType.JSON);
  }
}

function handleAddTasks(sheet, tasks) {
  const rows = [];
  
  tasks.forEach(task => {
    // Keep "Unsure" as-is, don't convert to empty string
    let dueDate = task.dueDate || 'Unsure';
    
    // Format timestamp as "DAY MONTH YEAR" (e.g., "28 May 2025 at 9:15 PM")
    const now = new Date();
    const day = now.getDate();
    const month = now.toLocaleDateString('en-US', { month: 'long' });
    const year = now.getFullYear();
    const time = now.toLocaleTimeString('en-US', { 
      hour: '2-digit', 
      minute: '2-digit', 
      hour12: true 
    });
    const formattedTimestamp = `${day} ${month} ${year} at ${time}`;
    
    rows.push([
      formattedTimestamp,                          // Timestamp in "28 May 2025 at 9:15 PM" format
      task.people ? task.people.join(', ') : '',
      task.client || 'unclear',
      task.summary || '',
      task.fullMessage || '',
      'Not Started',
      dueDate,
      task.botNotes || ''
    ]);
  });
  
  if (rows.length > 0) {
    const lastRow = sheet.getLastRow();
    sheet.getRange(lastRow + 1, 1, rows.length, 8).setValues(rows);
  }
  
  return ContentService
    .createTextOutput(JSON.stringify({
      status: 'success',
      rowsAdded: rows.length
    }))
    .setMimeType(ContentService.MimeType.JSON);
}

function handleGetTeam(sheet) {
  const data = sheet.getDataRange().getValues();
  const team = [];
  
  for (let i = 1; i < data.length; i++) {
    if (data[i][0] && data[i][1]) {
      team.push({
        name: data[i][0].toString().toLowerCase().trim(),
        email: data[i][1].toString().trim()
      });
    }
  }
  
  return ContentService
    .createTextOutput(JSON.stringify({
      status: 'success',
      team: team
    }))
    .setMimeType(ContentService.MimeType.JSON);
}

function handleGetTasks(sheet, statusFilter) {
  const data = sheet.getDataRange().getValues();
  const tasks = [];
  
  for (let i = 1; i < data.length; i++) {
    const status = data[i][5];
    
    if (!statusFilter || status !== statusFilter) {
      tasks.push({
        timestamp: data[i][0],
        people: data[i][1].toString().split(',').map(p => p.trim()),
        client: data[i][2],
        summary: data[i][3],
        fullMessage: data[i][4],
        status: data[i][5],
        dueDate: data[i][6],
        botNotes: data[i][7]
      });
    }
  }
  
  return ContentService
    .createTextOutput(JSON.stringify({
      status: 'success',
      tasks: tasks
    }))
    .setMimeType(ContentService.MimeType.JSON);
}

function initializeSheets() {
  const sheet = SpreadsheetApp.getActiveSpreadsheet();
  
  // Initialize todo sheet
  let todoSheet = sheet.getSheetByName('todo');
  if (!todoSheet) {
    todoSheet = sheet.insertSheet('todo');
  }
  
  const todoHeaders = ['Timestamp', 'People', 'Client', 'Summary', 'FullMessage', 'Status', 'DueDate', 'BotNotes'];
  todoSheet.getRange(1, 1, 1, todoHeaders.length).setValues([todoHeaders]);
  todoSheet.getRange(1, 1, 1, todoHeaders.length).setFontWeight('bold');
  
  // Add data validation for Status column (column F)
  const statusRule = SpreadsheetApp.newDataValidation()
    .requireValueInList(['Not Started', 'In Progress', 'Complete'], true)
    .setAllowInvalid(false)
    .setHelpText('Select task status')
    .build();
  
  // Apply validation to the entire Status column (starting from row 2)
  const lastRow = Math.max(todoSheet.getLastRow(), 100); // Ensure we cover at least 100 rows
  todoSheet.getRange(2, 6, lastRow - 1, 1).setDataValidation(statusRule);
  
  // Initialize team sheet
  let teamSheet = sheet.getSheetByName('team');
  if (!teamSheet) {
    teamSheet = sheet.insertSheet('team');
  }
  
  const teamHeaders = ['Name', 'Email'];
  teamSheet.getRange(1, 1, 1, teamHeaders.length).setValues([teamHeaders]);
  teamSheet.getRange(1, 1, 1, teamHeaders.length).setFontWeight('bold');
  
  const sampleTeam = [
    ['alice', 'alice@example.com'],
    ['bob', 'bob@example.com'],
    ['sarah', 'sarah@example.com']
  ];
  teamSheet.getRange(2, 1, sampleTeam.length, 2).setValues(sampleTeam);
  
  Logger.log('Sheets initialized successfully with Status dropdown!');
}

function testScript() {
  const testData = {
    postData: {
      contents: JSON.stringify({
        action: 'add_tasks',
        tasks: [{
          people: ['team'],
          client: 'Test',
          summary: 'Test task with readable timestamp',
          fullMessage: 'This is a test with new timestamp format',
          dueDate: 'Unsure',
          botNotes: 'Test with MONTH DAY YEAR format'
        }]
      })
    }
  };
  
  const result = doPost(testData);
  Logger.log(result.getContent());
}

/**
 * Run this function to fix the Status dropdown on existing sheets
 */
function fixStatusDropdown() {
  const sheet = SpreadsheetApp.getActiveSpreadsheet();
  const todoSheet = sheet.getSheetByName('todo');
  
  if (!todoSheet) {
    Logger.log('ERROR: todo sheet not found');
    return;
  }
  
  // Create the status validation rule
  const statusRule = SpreadsheetApp.newDataValidation()
    .requireValueInList(['Not Started', 'In Progress', 'Complete'], true)
    .setAllowInvalid(false)
    .setHelpText('Select task status')
    .build();
  
  // Apply to the entire Status column (column F, starting from row 2)
  const lastRow = Math.max(todoSheet.getLastRow(), 100);
  todoSheet.getRange(2, 6, lastRow - 1, 1).setDataValidation(statusRule);
  
  Logger.log('Status dropdown validation restored to column F (Status)!');
} 