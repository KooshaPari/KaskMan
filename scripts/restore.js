#!/usr/bin/env node

/**
 * Restore script for KaskManager R&D Platform
 * Restores data from backup files
 */

import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';
import { execSync } from 'child_process';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const BACKUP_DIR = process.env.BACKUP_DIR || path.join(__dirname, '../backups');
const DATA_DIR = process.env.RND_DATA_DIR || path.join(__dirname, '../data');

/**
 * List available backups
 */
async function listBackups() {
  try {
    const files = await fs.promises.readdir(BACKUP_DIR);
    const backupFiles = files
      .filter(file => file.endsWith('.tar.gz'))
      .map(file => ({
        name: file,
        path: path.join(BACKUP_DIR, file),
        mtime: fs.statSync(path.join(BACKUP_DIR, file)).mtime
      }))
      .sort((a, b) => b.mtime - a.mtime);

    return backupFiles;
  } catch (error) {
    console.error('❌ Failed to list backups:', error);
    return [];
  }
}

/**
 * Restore from backup file
 */
async function restoreFromBackup(backupFile, targetDir) {
  console.log(`🔄 Restoring from backup: ${backupFile}`);
  
  try {
    // Create target directory if it doesn't exist
    await fs.promises.mkdir(targetDir, { recursive: true });
    
    // Extract backup
    execSync(`tar -xzf "${backupFile}" -C "${path.dirname(targetDir)}"`, {
      stdio: 'inherit'
    });
    
    console.log(`✅ Restore completed: ${targetDir}`);
    return true;
  } catch (error) {
    console.error(`❌ Failed to restore: ${error.message}`);
    return false;
  }
}

/**
 * Interactive restore process
 */
async function interactiveRestore() {
  const backups = await listBackups();
  
  if (backups.length === 0) {
    console.log('⚠️  No backups found in:', BACKUP_DIR);
    return;
  }
  
  console.log('📋 Available backups:');
  backups.forEach((backup, index) => {
    console.log(`${index + 1}. ${backup.name} (${backup.mtime.toLocaleString()})`);
  });
  
  const { default: inquirer } = await import('inquirer');
  
  const answers = await inquirer.prompt([
    {
      type: 'list',
      name: 'backupIndex',
      message: 'Select backup to restore:',
      choices: backups.map((backup, index) => ({
        name: `${backup.name} (${backup.mtime.toLocaleString()})`,
        value: index
      }))
    },
    {
      type: 'confirm',
      name: 'confirm',
      message: 'This will overwrite existing data. Are you sure?',
      default: false
    }
  ]);
  
  if (!answers.confirm) {
    console.log('🚫 Restore cancelled');
    return;
  }
  
  const selectedBackup = backups[answers.backupIndex];
  return await restoreFromBackup(selectedBackup.path, DATA_DIR);
}

/**
 * Restore specific backup file
 */
async function restoreSpecific(backupName) {
  const backupPath = path.join(BACKUP_DIR, backupName);
  
  if (!fs.existsSync(backupPath)) {
    console.error(`❌ Backup file not found: ${backupPath}`);
    return false;
  }
  
  console.log(`🔄 Restoring specific backup: ${backupName}`);
  return await restoreFromBackup(backupPath, DATA_DIR);
}

/**
 * Main restore function
 */
async function main() {
  const args = process.argv.slice(2);
  
  if (args.length === 0) {
    // Interactive mode
    await interactiveRestore();
  } else {
    // Command line mode
    const command = args[0];
    
    switch (command) {
      case 'list':
        const backups = await listBackups();
        if (backups.length === 0) {
          console.log('⚠️  No backups found');
        } else {
          console.log('📋 Available backups:');
          backups.forEach(backup => {
            console.log(`  ${backup.name} (${backup.mtime.toLocaleString()})`);
          });
        }
        break;
        
      case 'restore':
        if (args.length < 2) {
          console.error('❌ Usage: npm run restore restore <backup-filename>');
          process.exit(1);
        }
        await restoreSpecific(args[1]);
        break;
        
      default:
        console.error('❌ Unknown command. Use: list, restore <filename>, or run without arguments for interactive mode');
        process.exit(1);
    }
  }
}

// Run restore if called directly
if (import.meta.url === `file://${process.argv[1]}`) {
  main().catch(console.error);
}

export { main as restore, listBackups, restoreFromBackup };