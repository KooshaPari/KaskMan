/**
 * Authentication Manager
 * Handles user authentication, authorization, and session management
 */

import { EventEmitter } from 'events';
import crypto from 'crypto';
import jwt from 'jsonwebtoken';
import bcrypt from 'bcrypt';
import { promises as fs } from 'fs';
import path from 'path';
import { Logger } from './logger.js';
import { ConfigManager } from './config-manager.js';

class AuthManager extends EventEmitter {
  constructor(config = {}) {
    super();
    this.config = {
      jwtSecret:
        config.jwtSecret || process.env.JWT_SECRET || this.generateSecret(),
      jwtExpiresIn: config.jwtExpiresIn || '24h',
      refreshTokenExpiresIn: config.refreshTokenExpiresIn || '7d',
      bcryptRounds: config.bcryptRounds || 12,
      sessionTimeout: config.sessionTimeout || 24 * 60 * 60 * 1000, // 24 hours
      maxLoginAttempts: config.maxLoginAttempts || 5,
      lockoutTime: config.lockoutTime || 15 * 60 * 1000, // 15 minutes
      usersFile: config.usersFile || './data/users.json',
      sessionsFile: config.sessionsFile || './data/sessions.json',
      ...config,
    };

    this.logger = new Logger('AuthManager');
    this.configManager = new ConfigManager();

    this.users = new Map();
    this.sessions = new Map();
    this.loginAttempts = new Map();
    this.refreshTokens = new Map();
    this.currentUser = null;
    this.currentSession = null;
  }

  async initialize() {
    try {
      await this.ensureDataDirectory();
      await this.loadUsers();
      await this.loadSessions();
      await this.createDefaultAdmin();

      // Cleanup expired sessions
      setInterval(() => this.cleanupExpiredSessions(), 60 * 60 * 1000); // Every hour

      this.logger.info('AuthManager initialized successfully');
    } catch (error) {
      this.logger.error('Failed to initialize AuthManager:', error);
      throw error;
    }
  }

  generateSecret() {
    return crypto.randomBytes(64).toString('hex');
  }

  async ensureDataDirectory() {
    const dataDir = path.dirname(this.config.usersFile);
    await fs.mkdir(dataDir, { recursive: true });
  }

  async loadUsers() {
    try {
      const usersData = await fs.readFile(this.config.usersFile, 'utf8');
      const users = JSON.parse(usersData);

      for (const user of users) {
        this.users.set(user.id, user);
      }

      this.logger.info(`Loaded ${this.users.size} users`);
    } catch (error) {
      if (error.code === 'ENOENT') {
        this.logger.info(
          'No users file found, starting with empty user database'
        );
        await this.saveUsers();
      } else {
        this.logger.error('Failed to load users:', error);
        throw error;
      }
    }
  }

  async saveUsers() {
    try {
      const users = Array.from(this.users.values());
      await fs.writeFile(this.config.usersFile, JSON.stringify(users, null, 2));
    } catch (error) {
      this.logger.error('Failed to save users:', error);
      throw error;
    }
  }

  async loadSessions() {
    try {
      const sessionsData = await fs.readFile(this.config.sessionsFile, 'utf8');
      const sessions = JSON.parse(sessionsData);

      for (const session of sessions) {
        // Only load non-expired sessions
        if (new Date(session.expiresAt) > new Date()) {
          this.sessions.set(session.id, session);
        }
      }

      this.logger.info(`Loaded ${this.sessions.size} active sessions`);
    } catch (error) {
      if (error.code === 'ENOENT') {
        this.logger.info(
          'No sessions file found, starting with empty session database'
        );
        await this.saveSessions();
      } else {
        this.logger.error('Failed to load sessions:', error);
        throw error;
      }
    }
  }

  async saveSessions() {
    try {
      const sessions = Array.from(this.sessions.values());
      await fs.writeFile(
        this.config.sessionsFile,
        JSON.stringify(sessions, null, 2)
      );
    } catch (error) {
      this.logger.error('Failed to save sessions:', error);
      throw error;
    }
  }

  async createDefaultAdmin() {
    // Create default admin user if no users exist
    if (this.users.size === 0) {
      const defaultAdmin = {
        id: crypto.randomUUID(),
        username: 'admin',
        email: 'admin@localhost',
        password: await bcrypt.hash('admin123', this.config.bcryptRounds),
        role: 'admin',
        permissions: ['*'],
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString(),
        active: true,
        lastLogin: null,
        loginAttempts: 0,
        lockedUntil: null,
      };

      this.users.set(defaultAdmin.id, defaultAdmin);
      await this.saveUsers();

      this.logger.info(
        'Created default admin user (username: admin, password: admin123)'
      );
      this.logger.warn('Please change the default admin password immediately!');
    }
  }

  async createUser(userData) {
    try {
      // Check if username already exists
      const existingUser = Array.from(this.users.values()).find(
        (u) => u.username === userData.username
      );
      if (existingUser) {
        throw new Error('Username already exists');
      }

      // Check if email already exists
      const existingEmail = Array.from(this.users.values()).find(
        (u) => u.email === userData.email
      );
      if (existingEmail) {
        throw new Error('Email already exists');
      }

      const user = {
        id: crypto.randomUUID(),
        username: userData.username,
        email: userData.email,
        password: await bcrypt.hash(
          userData.password,
          this.config.bcryptRounds
        ),
        role: userData.role || 'user',
        permissions: userData.permissions || ['read'],
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString(),
        active: userData.active !== false,
        lastLogin: null,
        loginAttempts: 0,
        lockedUntil: null,
        profile: userData.profile || {},
      };

      this.users.set(user.id, user);
      await this.saveUsers();

      this.emit('user:created', user);
      this.logger.info(`User created: ${user.username} (${user.id})`);

      // Return user without password
      const { password: _password, ...userWithoutPassword } = user;
      return userWithoutPassword;
    } catch (error) {
      this.logger.error('Failed to create user:', error);
      throw error;
    }
  }

  async login(credentials) {
    try {
      const { username, password, token } = credentials;

      // Token-based login
      if (token) {
        const decoded = jwt.verify(token, this.config.jwtSecret);
        const user = this.users.get(decoded.userId);

        if (!user || !user.active) {
          throw new Error('Invalid token or user not active');
        }

        const session = await this.createSession(user);
        this.currentUser = user;
        this.currentSession = session;

        this.emit('user:login', { user, session, method: 'token' });
        return { user: this.sanitizeUser(user), session, token };
      }

      // Username/password login
      if (!username || !password) {
        throw new Error('Username and password are required');
      }

      const user = Array.from(this.users.values()).find(
        (u) => u.username === username
      );
      if (!user) {
        throw new Error('Invalid username or password');
      }

      // Check if account is locked
      if (user.lockedUntil && new Date(user.lockedUntil) > new Date()) {
        throw new Error(
          'Account is locked due to too many failed login attempts'
        );
      }

      // Check login attempts
      if (user.loginAttempts >= this.config.maxLoginAttempts) {
        user.lockedUntil = new Date(
          Date.now() + this.config.lockoutTime
        ).toISOString();
        await this.saveUsers();
        throw new Error('Account locked due to too many failed login attempts');
      }

      // Verify password
      const isPasswordValid = await bcrypt.compare(password, user.password);
      if (!isPasswordValid) {
        user.loginAttempts = (user.loginAttempts || 0) + 1;
        await this.saveUsers();
        throw new Error('Invalid username or password');
      }

      // Check if user is active
      if (!user.active) {
        throw new Error('Account is disabled');
      }

      // Reset login attempts on successful login
      user.loginAttempts = 0;
      user.lockedUntil = null;
      user.lastLogin = new Date().toISOString();
      await this.saveUsers();

      // Create session
      const session = await this.createSession(user);
      this.currentUser = user;
      this.currentSession = session;

      this.emit('user:login', { user, session, method: 'password' });
      this.logger.info(`User logged in: ${user.username} (${user.id})`);

      return {
        user: this.sanitizeUser(user),
        session,
        token: session.token,
      };
    } catch (error) {
      this.logger.error('Login failed:', error);
      throw error;
    }
  }

  async logout(sessionId) {
    try {
      const session = this.sessions.get(sessionId || this.currentSession?.id);
      if (!session) {
        throw new Error('Session not found');
      }

      const user = this.users.get(session.userId);

      // Remove session
      this.sessions.delete(session.id);
      await this.saveSessions();

      // Remove refresh token
      this.refreshTokens.delete(session.refreshToken);

      // Clear current session if it's the one being logged out
      if (this.currentSession?.id === session.id) {
        this.currentUser = null;
        this.currentSession = null;
      }

      this.emit('user:logout', { user, session });
      this.logger.info(`User logged out: ${user?.username} (${user?.id})`);

      return { success: true };
    } catch (error) {
      this.logger.error('Logout failed:', error);
      throw error;
    }
  }

  async createSession(user) {
    const sessionId = crypto.randomUUID();
    const refreshToken = crypto.randomBytes(32).toString('hex');
    const expiresAt = new Date(
      Date.now() + this.config.sessionTimeout
    ).toISOString();

    const session = {
      id: sessionId,
      userId: user.id,
      refreshToken,
      createdAt: new Date().toISOString(),
      expiresAt,
      lastActivity: new Date().toISOString(),
      ipAddress: null, // Will be set by middleware
      userAgent: null, // Will be set by middleware
    };

    // Generate JWT token
    const token = jwt.sign(
      {
        sessionId: session.id,
        userId: user.id,
        username: user.username,
        role: user.role,
        permissions: user.permissions,
      },
      this.config.jwtSecret,
      { expiresIn: this.config.jwtExpiresIn }
    );

    session.token = token;

    this.sessions.set(sessionId, session);
    this.refreshTokens.set(refreshToken, sessionId);
    await this.saveSessions();

    return session;
  }

  async refreshSession(refreshToken) {
    try {
      const sessionId = this.refreshTokens.get(refreshToken);
      if (!sessionId) {
        throw new Error('Invalid refresh token');
      }

      const session = this.sessions.get(sessionId);
      if (!session) {
        throw new Error('Session not found');
      }

      if (new Date(session.expiresAt) < new Date()) {
        throw new Error('Session expired');
      }

      const user = this.users.get(session.userId);
      if (!user || !user.active) {
        throw new Error('User not found or inactive');
      }

      // Create new session
      const newSession = await this.createSession(user);

      // Remove old session
      this.sessions.delete(sessionId);
      this.refreshTokens.delete(refreshToken);
      await this.saveSessions();

      this.emit('session:refreshed', { user, oldSession: session, newSession });

      return {
        user: this.sanitizeUser(user),
        session: newSession,
        token: newSession.token,
      };
    } catch (error) {
      this.logger.error('Session refresh failed:', error);
      throw error;
    }
  }

  async verifyToken(token) {
    try {
      const decoded = jwt.verify(token, this.config.jwtSecret);

      const session = this.sessions.get(decoded.sessionId);
      if (!session) {
        throw new Error('Session not found');
      }

      if (new Date(session.expiresAt) < new Date()) {
        throw new Error('Session expired');
      }

      const user = this.users.get(decoded.userId);
      if (!user || !user.active) {
        throw new Error('User not found or inactive');
      }

      // Update last activity
      session.lastActivity = new Date().toISOString();
      await this.saveSessions();

      return {
        user: this.sanitizeUser(user),
        session,
        decoded,
      };
    } catch (error) {
      if (error.name === 'TokenExpiredError') {
        throw new Error('Token expired');
      } else if (error.name === 'JsonWebTokenError') {
        throw new Error('Invalid token');
      }
      throw error;
    }
  }

  async getStatus() {
    return {
      authenticated: !!this.currentUser,
      user: this.currentUser ? this.sanitizeUser(this.currentUser) : null,
      session: this.currentSession,
      expiresAt: this.currentSession?.expiresAt,
    };
  }

  async requireAuth() {
    if (!this.currentUser) {
      throw new Error('Authentication required');
    }
    return this.currentUser;
  }

  async checkPermission(permission) {
    if (!this.currentUser) {
      throw new Error('Authentication required');
    }

    // Admin has all permissions
    if (this.currentUser.role === 'admin') {
      return true;
    }

    // Check specific permissions
    return (
      this.currentUser.permissions.includes(permission) ||
      this.currentUser.permissions.includes('*')
    );
  }

  async requirePermission(permission) {
    if (!(await this.checkPermission(permission))) {
      throw new Error(`Permission denied: ${permission}`);
    }
  }

  sanitizeUser(user) {
    const { password: _password, ...sanitizedUser } = user;
    return sanitizedUser;
  }

  async cleanupExpiredSessions() {
    const now = new Date();
    const expiredSessions = [];

    for (const [sessionId, session] of this.sessions.entries()) {
      if (new Date(session.expiresAt) < now) {
        expiredSessions.push(sessionId);
      }
    }

    for (const sessionId of expiredSessions) {
      const session = this.sessions.get(sessionId);
      this.sessions.delete(sessionId);
      this.refreshTokens.delete(session.refreshToken);
    }

    if (expiredSessions.length > 0) {
      await this.saveSessions();
      this.logger.info(`Cleaned up ${expiredSessions.length} expired sessions`);
    }
  }

  // User management methods
  async listUsers() {
    return Array.from(this.users.values()).map((user) =>
      this.sanitizeUser(user)
    );
  }

  async getUser(userId) {
    const user = this.users.get(userId);
    if (!user) {
      throw new Error('User not found');
    }
    return this.sanitizeUser(user);
  }

  async updateUser(userId, updates) {
    const user = this.users.get(userId);
    if (!user) {
      throw new Error('User not found');
    }

    // Hash password if provided
    if (updates.password) {
      updates.password = await bcrypt.hash(
        updates.password,
        this.config.bcryptRounds
      );
    }

    const updatedUser = {
      ...user,
      ...updates,
      updatedAt: new Date().toISOString(),
    };

    this.users.set(userId, updatedUser);
    await this.saveUsers();

    this.emit('user:updated', updatedUser);
    this.logger.info(`User updated: ${updatedUser.username} (${userId})`);

    return this.sanitizeUser(updatedUser);
  }

  async deleteUser(userId) {
    const user = this.users.get(userId);
    if (!user) {
      throw new Error('User not found');
    }

    // Don't allow deleting the last admin
    if (user.role === 'admin') {
      const adminCount = Array.from(this.users.values()).filter(
        (u) => u.role === 'admin'
      ).length;
      if (adminCount <= 1) {
        throw new Error('Cannot delete the last admin user');
      }
    }

    // Remove user sessions
    const userSessions = Array.from(this.sessions.values()).filter(
      (s) => s.userId === userId
    );
    for (const session of userSessions) {
      this.sessions.delete(session.id);
      this.refreshTokens.delete(session.refreshToken);
    }

    this.users.delete(userId);
    await this.saveUsers();
    await this.saveSessions();

    this.emit('user:deleted', user);
    this.logger.info(`User deleted: ${user.username} (${userId})`);

    return this.sanitizeUser(user);
  }

  async changePassword(userId, oldPassword, newPassword) {
    const user = this.users.get(userId);
    if (!user) {
      throw new Error('User not found');
    }

    // Verify old password
    const isOldPasswordValid = await bcrypt.compare(oldPassword, user.password);
    if (!isOldPasswordValid) {
      throw new Error('Current password is incorrect');
    }

    // Hash new password
    const hashedNewPassword = await bcrypt.hash(
      newPassword,
      this.config.bcryptRounds
    );

    // Update user
    user.password = hashedNewPassword;
    user.updatedAt = new Date().toISOString();

    this.users.set(userId, user);
    await this.saveUsers();

    this.emit('user:password-changed', user);
    this.logger.info(`Password changed for user: ${user.username} (${userId})`);

    return { success: true };
  }

  async stop() {
    try {
      await this.saveSessions();
      await this.saveUsers();

      this.logger.info('AuthManager stopped successfully');
    } catch (error) {
      this.logger.error('Error stopping AuthManager:', error);
      throw error;
    }
  }
}

export { AuthManager };
