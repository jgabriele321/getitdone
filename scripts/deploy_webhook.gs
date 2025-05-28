/**
 * Google Apps Script Webhook for Go TODO Bot
 * 
 * This script receives POST requests from the Go bot and writes tasks to Google Sheets.
 * 
 * Setup:
 * 1. Create a Google Sheet with two tabs: "todo" and "team"
 * 2. Open Extensions → Apps Script
 * 3. Paste this code
 * 4. Deploy → New Deployment → Web app
 * 5. Execute as: Me, Who has access: Anyone
 * 6. Copy the deployment URL for GOOGLE_SCRIPT_URL
 */

/**
 * Handles POST requests from the bot
 * Enhanced version with comprehensive debugging and redirect handling
 */
function doPost(e) {
  // Start with comprehensive logging
  Logger.log("=== doPost Function Started ===");
  Logger.log("Timestamp: " + new Date().toISOString());
  
  // Log the raw event object
  Logger.log("Raw event object (e): " + JSON.stringify(e));
  Logger.log("Event object type: " + typeof e);
  Logger.log("Event object keys: " + (e ? Object.keys(e) : "e is null/undefined"));
  
  // Check if e exists at all
  if (!e) {
    Logger.log("CRITICAL: Event object 'e' is null or undefined");
    return createErrorResponse("Event object is null or undefined", "MISSING_EVENT");
  }
  
  // Log all properties of e
  for (let key in e) {
    try {
      Logger.log("e." + key + ": " + JSON.stringify(e[key]));
    } catch (err) {
      Logger.log("e." + key + ": [Cannot stringify - " + err.message + "]");
    }
  }
  
  // Check postData specifically
  Logger.log("e.postData exists: " + (e.postData !== undefined));
  Logger.log("e.postData type: " + typeof e.postData);
  
  if (e.postData) {
    Logger.log("e.postData keys: " + Object.keys(e.postData));
    Logger.log("e.postData.contents exists: " + (e.postData.contents !== undefined));
    Logger.log("e.postData.contents type: " + typeof e.postData.contents);
    Logger.log("e.postData.contents value: " + e.postData.contents);
    Logger.log("e.postData.length: " + e.postData.length);
    Logger.log("e.postData.type: " + e.postData.type);
  }
  
  // Check for alternative data sources (query parameters, etc.)
  Logger.log("e.parameter: " + JSON.stringify(e.parameter));
  Logger.log("e.parameters: " + JSON.stringify(e.parameters));
  Logger.log("e.queryString: " + e.queryString);
  
  try {
    let requestData = null;
    let dataSource = "unknown";
    
    // Try multiple ways to get the request data
    if (e.postData && e.postData.contents) {
      // Standard POST data
      requestData = e.postData.contents;
      dataSource = "postData.contents";
      Logger.log("Using postData.contents as data source");
    } else if (e.postData && e.postData.getDataAsString) {
      // Alternative method for getting POST data
      requestData = e.postData.getDataAsString();
      dataSource = "postData.getDataAsString()";
      Logger.log("Using postData.getDataAsString() as data source");
    } else if (e.parameter && Object.keys(e.parameter).length > 0) {
      // Fallback to query parameters (might happen with redirects)
      requestData = JSON.stringify(e.parameter);
      dataSource = "parameter (query params)";
      Logger.log("Using query parameters as data source");
    } else if (e.queryString) {
      // Try to parse query string
      requestData = e.queryString;
      dataSource = "queryString";
      Logger.log("Using query string as data source");
    }
    
    Logger.log("Data source used: " + dataSource);
    Logger.log("Raw request data: " + requestData);
    
    if (!requestData) {
      Logger.log("ERROR: No request data found in any expected location");
      return createErrorResponse("No request data found", "NO_DATA", {
        eventKeys: e ? Object.keys(e) : [],
        postDataExists: !!e.postData,
        parameterExists: !!e.parameter,
        queryStringExists: !!e.queryString
      });
    }
    
    // Try to parse the data as JSON
    let data;
    try {
      // If it's already an object (from parameters), use it directly
      if (typeof requestData === 'object') {
        data = requestData;
      } else {
        data = JSON.parse(requestData);
      }
      Logger.log("Successfully parsed request data: " + JSON.stringify(data));
    } catch (parseError) {
      Logger.log("JSON parse error: " + parseError.message);
      Logger.log("Attempting to handle as form data or query string...");
      
      // Try to handle as URL-encoded form data
      if (typeof requestData === 'string' && requestData.includes('=')) {
        data = parseFormData(requestData);
        Logger.log("Parsed as form data: " + JSON.stringify(data));
      } else {
        throw new Error("Unable to parse request data as JSON or form data: " + parseError.message);
      }
    }
    
    // Validate that we have an action
    if (!data.action) {
      Logger.log("ERROR: No action specified in request data");
      return createErrorResponse("No action specified", "MISSING_ACTION", { receivedData: data });
    }
    
    Logger.log("Processing action: " + data.action);
    
    // Open the spreadsheet
    const sheet = SpreadsheetApp.getActiveSpreadsheet();
    const todoSheet = sheet.getSheetByName('todo');
    const teamSheet = sheet.getSheetByName('team');
    
    // Validate sheets exist
    if (!todoSheet || !teamSheet) {
      Logger.log('ERROR: Missing required sheets. "todo" or "team" tabs not found.');
      return createErrorResponse('Missing required sheets. Please create "todo" and "team" tabs.', "MISSING_SHEETS");
    }
    Logger.log('Sheets "todo" and "team" accessed successfully.');
    
    // Handle different request types
    let result;
    if (data.action === 'add_tasks') {
      Logger.log('Processing add_tasks action');
      result = handleAddTasks(todoSheet, data.tasks);
    } else if (data.action === 'get_team') {
      Logger.log('Processing get_team action');
      result = handleGetTeam(teamSheet);
    } else if (data.action === 'get_tasks') {
      Logger.log('Processing get_tasks action');
      result = handleGetTasks(todoSheet, data.status);
    } else {
      Logger.log('ERROR: Unknown action received - ' + data.action);
      return createErrorResponse('Unknown action: ' + data.action, "UNKNOWN_ACTION", { receivedAction: data.action });
    }
    
    Logger.log("Action completed successfully");
    return result;
    
  } catch (error) {
    // Comprehensive error logging
    Logger.log('=== ERROR in doPost ===');
    Logger.log('Error name: ' + error.name);
    Logger.log('Error message: ' + error.message);
    Logger.log('Error stack: ' + error.stack);
    Logger.log('Error toString: ' + error.toString());
    
    return createErrorResponse(error.message, error.name, {
      stack: error.stack,
      eventObjectExists: !!e,
      postDataExists: !!(e && e.postData),
      timestamp: new Date().toISOString()
    });
  }
}

/**
 * Helper function to create consistent error responses
 */
function createErrorResponse(message, errorType, additionalData) {
  const errorResponse = {
    status: 'error',
    message: message,
    errorType: errorType || 'UNKNOWN_ERROR',
    timestamp: new Date().toISOString()
  };
  
  if (additionalData) {
    errorResponse.debug = additionalData;
  }
  
  Logger.log("Returning error response: " + JSON.stringify(errorResponse));
  
  return ContentService
    .createTextOutput(JSON.stringify(errorResponse))
    .setMimeType(ContentService.MimeType.JSON);
}

/**
 * Helper function to parse form-encoded data
 */
function parseFormData(formString) {
  const result = {};
  const pairs = formString.split('&');
  
  for (let pair of pairs) {
    const [key, value] = pair.split('=');
    if (key && value) {
      result[decodeURIComponent(key)] = decodeURIComponent(value);
    }
  }
  
  return result;
}

/**
 * Handles adding new tasks to the todo sheet
 */
function handleAddTasks(sheet, tasks) {
  const rows = [];
  
  // Process each task
  tasks.forEach(task => {
    // Handle dueDate - convert "Unsure" to empty string for date validation
    let dueDate = task.dueDate || '';
    if (dueDate === 'Unsure' || dueDate === 'unclear') {
      dueDate = ''; // Empty string passes date validation
    }
    
    rows.push([
      new Date().toISOString(),                    // Timestamp
      task.people ? task.people.join(', ') : '',   // People (comma-separated)
      task.client || 'unclear',                    // Client (default to unclear)
      task.summary || '',                          // Summary
      task.fullMessage || '',                      // FullMessage
      'Not Started',                               // Status (default)
      dueDate,                                     // DueDate (empty if "Unsure")
      task.botNotes || ''                          // BotNotes
    ]);
  });
  
  // Append all rows at once
  if (rows.length > 0) {
    const lastRow = sheet.getLastRow();
    sheet.getRange(lastRow + 1, 1, rows.length, 8).setValues(rows);
  }
  
  return ContentService
    .createTextOutput(JSON.stringify({
      status: 'success',
      rowsAdded: rows.length,
      message: `Added ${rows.length} task(s)`
    }))
    .setMimeType(ContentService.MimeType.JSON);
}

/**
 * Gets team members from the team sheet
 */
function handleGetTeam(sheet) {
  const data = sheet.getDataRange().getValues();
  const team = [];
  
  // Skip header row
  for (let i = 1; i < data.length; i++) {
    if (data[i][0] && data[i][1]) {  // Ensure both name and email exist
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

/**
 * Gets tasks filtered by status
 */
function handleGetTasks(sheet, statusFilter) {
  const data = sheet.getDataRange().getValues();
  const tasks = [];
  
  // Skip header row
  for (let i = 1; i < data.length; i++) {
    const status = data[i][5]; // Status column (now at index 5)
    
    // Filter by status if provided
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

/**
 * Initialize sheets with proper headers and data validation
 * Run this once after creating the spreadsheet
 */
function initializeSheets() {
  const sheet = SpreadsheetApp.getActiveSpreadsheet();
  
  // Initialize todo sheet
  let todoSheet = sheet.getSheetByName('todo');
  if (!todoSheet) {
    todoSheet = sheet.insertSheet('todo');
  }
  
  // Set headers for todo sheet
  const todoHeaders = ['Timestamp', 'People', 'Client', 'Summary', 'FullMessage', 'Status', 'DueDate', 'BotNotes'];
  todoSheet.getRange(1, 1, 1, todoHeaders.length).setValues([todoHeaders]);
  todoSheet.getRange(1, 1, 1, todoHeaders.length).setFontWeight('bold');
  
  // Add data validation for Status column
  const statusRule = SpreadsheetApp.newDataValidation()
    .requireValueInList(['Not Started', 'In Progress', 'Complete'], true)
    .setAllowInvalid(false)
    .build();
  todoSheet.getRange('F2:F').setDataValidation(statusRule);

  // Add date format validation for DueDate column
  const dueDateRule = SpreadsheetApp.newDataValidation()
    .requireDate()
    .setAllowInvalid(false)
    .build();
  todoSheet.getRange('G2:G').setDataValidation(dueDateRule);
  
  // Initialize team sheet
  let teamSheet = sheet.getSheetByName('team');
  if (!teamSheet) {
    teamSheet = sheet.insertSheet('team');
  }
  
  // Set headers for team sheet
  const teamHeaders = ['Name', 'Email'];
  teamSheet.getRange(1, 1, 1, teamHeaders.length).setValues([teamHeaders]);
  teamSheet.getRange(1, 1, 1, teamHeaders.length).setFontWeight('bold');
  
  // Add sample team members
  const sampleTeam = [
    ['alice', 'alice@example.com'],
    ['bob', 'bob@example.com'],
    ['sarah', 'sarah@example.com']
  ];
  teamSheet.getRange(2, 1, sampleTeam.length, 2).setValues(sampleTeam);
  
  // Auto-resize columns
  todoSheet.autoResizeColumns(1, todoHeaders.length);
  teamSheet.autoResizeColumns(1, teamHeaders.length);
  
  Logger.log('Sheets initialized successfully!');
}

/**
 * Test function to verify the script is working
 */
function testScript() {
  const testData = {
    postData: {
      contents: JSON.stringify({
        action: 'add_tasks',
        tasks: [{
          people: ['alice', 'bob'],
          client: 'Microsoft',
          summary: 'Test task from Apps Script',
          fullMessage: 'This is a test task created by the test function',
          dueDate: '2024-03-15',
          botNotes: 'Test successful'
        }]
      })
    }
  };
  
  const result = doPost(testData);
  Logger.log(result.getContent());
}

function doGet(e) {
  Logger.log("doGet triggered. Event (e): " + JSON.stringify(e));
  return ContentService.createTextOutput("Script is reachable via GET. Hello from doGet!").setMimeType(ContentService.MimeType.TEXT);
} 