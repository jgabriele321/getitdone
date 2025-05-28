/**
 * Google Apps Script Webhook for TODO Bot
 * Handles add_tasks, get_team, and get_tasks actions
 */

function doPost(e) {
  try {
    const data = JSON.parse(e.postData.contents);
    const action = data.action;
    
    switch (action) {
      case 'add_tasks':
        return addTasks(data.tasks);
      case 'get_team':
        return getTeam();
      case 'get_tasks':
        return getTasks(data.person);
      default:
        return createResponse(false, 'Unknown action: ' + action);
    }
  } catch (error) {
    console.error('Error in doPost:', error);
    return createResponse(false, 'Error: ' + error.toString());
  }
}

function addTasks(tasks) {
  try {
    const sheet = getOrCreateSheet('todo');
    
    // Ensure headers exist
    ensureTodoHeaders(sheet);
    
    // Add each task as a row
    tasks.forEach(task => {
      const row = [
        new Date().toISOString(), // Timestamp
        Array.isArray(task.people) ? task.people.join(', ') : task.people, // People
        task.client || 'Internal', // Client
        task.summary || '', // Summary
        task.fullMessage || '', // FullMessage
        task.status || 'Not Started', // Status
        task.dueDate || 'unclear', // DueDate
        task.botNotes || `Confidence: ${task.confidence || 0.5}` // BotNotes
      ];
      sheet.appendRow(row);
    });
    
    return ContentService
      .createTextOutput(JSON.stringify({
        status: "success",
        rowsAdded: tasks.length,
        message: `Added ${tasks.length} task(s)`
      }))
      .setMimeType(ContentService.MimeType.JSON);
  } catch (error) {
    console.error('Error in addTasks:', error);
    return ContentService
      .createTextOutput(JSON.stringify({
        status: "error",
        error: error.toString()
      }))
      .setMimeType(ContentService.MimeType.JSON);
  }
}

function getTeam() {
  try {
    const sheet = getOrCreateSheet('team');
    
    // Ensure headers exist
    ensureTeamHeaders(sheet);
    
    const data = sheet.getDataRange().getValues();
    const headers = data[0];
    const rows = data.slice(1);
    
    const team = rows.map(row => {
      const member = {};
      headers.forEach((header, index) => {
        member[header.toLowerCase()] = row[index] || '';
      });
      return member;
    }).filter(member => member.name && member.email);
    
    return ContentService
      .createTextOutput(JSON.stringify({
        status: "success",
        team: team
      }))
      .setMimeType(ContentService.MimeType.JSON);
  } catch (error) {
    console.error('Error in getTeam:', error);
    return ContentService
      .createTextOutput(JSON.stringify({
        status: "error",
        error: error.toString()
      }))
      .setMimeType(ContentService.MimeType.JSON);
  }
}

function getTasks(person) {
  try {
    const sheet = getOrCreateSheet('todo');
    
    if (sheet.getLastRow() <= 1) {
      return createResponse(true, 'No tasks found', { tasks: [] });
    }
    
    const data = sheet.getDataRange().getValues();
    const headers = data[0];
    const rows = data.slice(1);
    
    const tasks = rows.map(row => {
      const task = {};
      headers.forEach((header, index) => {
        task[header.toLowerCase().replace(/\s+/g, '')] = row[index] || '';
      });
      return task;
    }).filter(task => {
      if (!person) return true;
      const people = task.people ? task.people.toLowerCase() : '';
      return people.includes(person.toLowerCase()) || people.includes('team');
    });
    
    return createResponse(true, 'Tasks retrieved', { tasks });
  } catch (error) {
    console.error('Error in getTasks:', error);
    return createResponse(false, 'Error getting tasks: ' + error.toString());
  }
}

function getOrCreateSheet(name) {
  const spreadsheet = SpreadsheetApp.getActiveSpreadsheet();
  let sheet = spreadsheet.getSheetByName(name);
  
  if (!sheet) {
    sheet = spreadsheet.insertSheet(name);
  }
  
  return sheet;
}

function ensureTodoHeaders(sheet) {
  if (sheet.getLastRow() === 0) {
    const headers = [
      'Timestamp',
      'People', 
      'Client',
      'Summary',
      'DueDate',
      'FullMessage',
      'Status',
      'BotNotes'
    ];
    sheet.getRange(1, 1, 1, headers.length).setValues([headers]);
    
    // Add data validation for Status column
    const statusRange = sheet.getRange(2, 7, 1000, 1); // Status column
    const statusRule = SpreadsheetApp.newDataValidation()
      .requireValueInList(['Not Started', 'In Progress', 'Complete'])
      .setAllowInvalid(false)
      .build();
    statusRange.setDataValidation(statusRule);
    
    // Format headers
    const headerRange = sheet.getRange(1, 1, 1, headers.length);
    headerRange.setFontWeight('bold');
    headerRange.setBackground('#f0f0f0');
  }
}

function ensureTeamHeaders(sheet) {
  if (sheet.getLastRow() === 0) {
    const headers = ['Name', 'Email'];
    sheet.getRange(1, 1, 1, headers.length).setValues([headers]);
    
    // Format headers
    const headerRange = sheet.getRange(1, 1, 1, headers.length);
    headerRange.setFontWeight('bold');
    headerRange.setBackground('#f0f0f0');
    
    // Add sample data
    sheet.appendRow(['team', 'team@example.com']);
    sheet.appendRow(['lilly', 'lilly@example.com']);
    sheet.appendRow(['jemma', 'jemma@example.com']);
  }
}

function createResponse(success, message, data = null) {
  const response = {
    success: success,
    message: message
  };
  
  if (data) {
    Object.assign(response, data);
  }
  
  return ContentService
    .createTextOutput(JSON.stringify(response))
    .setMimeType(ContentService.MimeType.JSON);
} 