#!/usr/bin/env node

/**
 * Backup script for KaskManager R&D Platform
 * Creates backups of data, configuration, and project files
 */

import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';
import { createReadStream, createWriteStream } from 'fs';
import { pipeline } from 'stream/promises';
import { createGzip } from 'zlib';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const BACKUP_DIR = process.env.BACKUP_DIR || path.join(__dirname, '../backups');
const DATA_DIR = process.env.RND_DATA_DIR || path.join(__dirname, '../data');

/**
 * Create backup directory if it doesn't exist
 */
async function ensureBackupDir() {
  try {
    await fs.promises.mkdir(BACKUP_DIR, { recursive: true });
  } catch (error) {
    if (error.code !== 'EEXIST') {
      throw error;
    }
  }
}

/**
 * Create compressed backup of a directory
 */
async function createBackup(sourceDir, backupName) {
  const timestamp = new Date().toISOString().replace(/[:.]/g, '-');
  const backupFileName = `${backupName}-${timestamp}.tar.gz`;
  const backupPath = path.join(BACKUP_DIR, backupFileName);

  console.log(`Creating backup: ${backupFileName}`);
  
  try {
    // Create tar.gz archive
    const { execSync } = await import('child_process');
    execSync(`tar -czf "${backupPath}" -C "${path.dirname(sourceDir)}" "${path.basename(sourceDir)}"`, {
      stdio: 'inherit'
    });
    
    console.log(`✅ Backup created: ${backupPath}`);
    return backupPath;
  } catch (error) {
    console.error(`❌ Failed to create backup: ${error.message}`);
    throw error;
  }
}

/**
 * Backup configuration files
 */
async function backupConfig() {
  const configFiles = [
    '.env',
    'package.json',
    'docker-compose.yml',
    'CLAUDE.md'
  ];

  const configBackupDir = path.join(BACKUP_DIR, 'config');
  await fs.promises.mkdir(configBackupDir, { recursive: true });

  for (const file of configFiles) {
    const sourceFile = path.join(__dirname, '..', file);
    const backupFile = path.join(configBackupDir, file);

    try {
      if (fs.existsSync(sourceFile)) {
        await fs.promises.copyFile(sourceFile, backupFile);
        console.log(`✅ Config backed up: ${file}`);
      }
    } catch (error) {
      console.warn(`⚠️  Could not backup config file ${file}: ${error.message}`);
    }
  }
}

/**
 * Clean old backups (keep last 10)
 */
async function cleanOldBackups() {
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

    if (backupFiles.length > 10) {
      const filesToDelete = backupFiles.slice(10);
      
      for (const file of filesToDelete) {
        await fs.promises.unlink(file.path);
        console.log(`🗑️  Removed old backup: ${file.name}`);
      }
    }
  } catch (error) {
    console.warn(`⚠️  Could not clean old backups: ${error.message}`);
  }
}

/**
 * Main backup function
 */
async function main() {
  console.log('🔄 Starting backup process...');
  
  try {
    await ensureBackupDir();
    
    // Backup data directory
    if (fs.existsSync(DATA_DIR)) {
      await createBackup(DATA_DIR, 'data');
    } else {
      console.log('⚠️  Data directory not found, skipping data backup');
    }
    
    // Backup configuration files
    await backupConfig();
    
    // Clean old backups
    await cleanOldBackups();
    
    console.log('✅ Backup process completed successfully');
    
    return {
      success: true,
      timestamp: new Date().toISOString(),
      backupDir: BACKUP_DIR
    };
    
  } catch (error) {
    console.error('❌ Backup process failed:', error);
    process.exit(1);
  }
}

// Run backup if called directly
if (import.meta.url === `file://${process.argv[1]}`) {
  main().catch(console.error);
}

export { main as backup };