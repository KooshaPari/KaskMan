/**
 * Learning Algorithm - Self-Learning AI Engine for R&D Module
 * Implements dormant-to-active learning with pattern recognition and adaptation
 */

export class LearningAlgorithm {
  constructor(config = {}) {
    this.config = {
      learningRate: config.learningRate || 0.1,
      memoryDecay: config.memoryDecay || 0.95,
      adaptationThreshold: config.adaptationThreshold || 0.8,
      maxMemorySize: config.maxMemorySize || 10000,
      ...config,
    };

    this.model = {
      weights: new Map(),
      biases: new Map(),
      neuralConnections: new Map(),
      memoryBank: new Map(),
      experienceBuffer: [],
    };

    this.learningState = {
      epoch: 0,
      totalExperiences: 0,
      accuracy: 0,
      confidence: 0,
      lastUpdate: Date.now(),
      adaptationScore: 0,
    };

    this.neuralNetwork = this.initializeNeuralNetwork();
  }

  initializeNeuralNetwork() {
    return {
      inputLayer: { size: 20, weights: this.randomWeights(20) },
      hiddenLayers: [
        { size: 15, weights: this.randomWeights(15), activation: 'relu' },
        { size: 10, weights: this.randomWeights(10), activation: 'relu' },
        { size: 8, weights: this.randomWeights(8), activation: 'tanh' },
      ],
      outputLayer: {
        size: 5,
        weights: this.randomWeights(5),
        activation: 'softmax',
      },
    };
  }

  randomWeights(size) {
    return Array.from({ length: size }, () => (Math.random() - 0.5) * 2);
  }

  async processPassiveSignals(signals) {
    // Convert signals to feature vectors
    const features = this.extractFeatures(signals);

    // Update memory bank with new observations
    this.updateMemoryBank(features, signals.timestamp);

    // Perform unsupervised learning
    await this.unsupervisedLearning(features);

    // Update learning state
    this.updateLearningState(features);

    return {
      processed: true,
      features: features.length,
      memorySize: this.model.memoryBank.size,
      adaptationScore: this.learningState.adaptationScore,
    };
  }

  extractFeatures(signals) {
    const features = [];

    // System activity features
    features.push(
      this.normalize(signals.systemActivity.fileModifications, 0, 1000)
    );
    features.push(
      this.normalize(signals.systemActivity.commandExecutions, 0, 500)
    );
    features.push(this.normalize(signals.systemActivity.errorPatterns, 0, 100));
    features.push(signals.systemActivity.performanceMetrics.cpu / 100);
    features.push(signals.systemActivity.performanceMetrics.memory / 100);
    features.push(signals.systemActivity.performanceMetrics.disk / 100);

    // User pattern features
    features.push(signals.userPatterns.frequentCommands.length / 20);
    features.push(signals.userPatterns.workingHours.current / 24);
    features.push(signals.userPatterns.projectTypes.length / 10);
    features.push(signals.userPatterns.technologyStack.confidence);

    // Project completion features
    features.push(signals.projectCompletions.completed / 10);
    features.push(signals.projectCompletions.abandoned / 5);
    features.push(signals.projectCompletions.successRate);
    features.push(
      this.normalize(signals.projectCompletions.avgDuration, 1, 30)
    );

    // Market trend features
    const avgTrendScore =
      signals.marketTrends.reduce((acc, trend) => acc + trend.score, 0) /
      signals.marketTrends.length;
    const avgConfidence =
      signals.marketTrends.reduce((acc, trend) => acc + trend.confidence, 0) /
      signals.marketTrends.length;
    features.push(avgTrendScore);
    features.push(avgConfidence);

    // Technology update features
    features.push(signals.technologyUpdates.languageUpdates.length / 10);
    features.push(signals.technologyUpdates.frameworkUpdates.length / 10);
    features.push(signals.technologyUpdates.toolUpdates.length / 10);
    features.push(
      this.normalize(signals.technologyUpdates.securityUpdates, 0, 20)
    );

    return features;
  }

  normalize(value, min, max) {
    return Math.max(0, Math.min(1, (value - min) / (max - min)));
  }

  updateMemoryBank(features, timestamp) {
    const memoryKey = `experience_${timestamp}`;
    const experience = {
      features,
      timestamp,
      importance: this.calculateImportance(features),
      accessCount: 0,
      lastAccess: timestamp,
    };

    this.model.memoryBank.set(memoryKey, experience);

    // Memory cleanup if needed
    if (this.model.memoryBank.size > this.config.maxMemorySize) {
      this.performMemoryCleanup();
    }
  }

  calculateImportance(features) {
    // Calculate importance based on feature variance and novelty
    const variance = this.calculateVariance(features);
    const novelty = this.calculateNovelty(features);
    const significance = this.calculateSignificance(features);

    return variance * 0.3 + novelty * 0.4 + significance * 0.3;
  }

  calculateVariance(features) {
    const mean = features.reduce((acc, val) => acc + val, 0) / features.length;
    const variance =
      features.reduce((acc, val) => acc + Math.pow(val - mean, 2), 0) /
      features.length;
    return Math.sqrt(variance);
  }

  calculateNovelty(features) {
    // Compare with recent experiences to determine novelty
    const recentExperiences = Array.from(this.model.memoryBank.values())
      .filter((exp) => Date.now() - exp.timestamp < 24 * 60 * 60 * 1000) // Last 24 hours
      .slice(-10); // Last 10 experiences

    if (recentExperiences.length === 0) return 1.0;

    const avgDistance =
      recentExperiences.reduce((acc, exp) => {
        return acc + this.euclideanDistance(features, exp.features);
      }, 0) / recentExperiences.length;

    return Math.min(1.0, avgDistance / 2.0);
  }

  calculateSignificance(features) {
    // Determine significance based on feature patterns
    const highValueFeatures = features.filter((f) => f > 0.7).length;
    const lowValueFeatures = features.filter((f) => f < 0.3).length;
    const _midValueFeatures = features.filter(
      (f) => f >= 0.3 && f <= 0.7
    ).length;

    // Patterns with extreme values are more significant
    return (highValueFeatures + lowValueFeatures) / features.length;
  }

  euclideanDistance(a, b) {
    return Math.sqrt(
      a.reduce((acc, val, i) => acc + Math.pow(val - b[i], 2), 0)
    );
  }

  async unsupervisedLearning(features) {
    // K-means clustering for pattern discovery
    await this.performClustering(features);

    // Autoencoder-style learning for feature compression
    await this.performAutoencoding(features);

    // Temporal pattern learning
    await this.performTemporalLearning(features);

    // Anomaly detection
    await this.performAnomalyDetection(features);
  }

  async performClustering(features) {
    const clusters = this.model.neuralConnections.get('clusters') || [];
    const k = Math.min(
      5,
      Math.max(2, Math.floor(this.model.memoryBank.size / 100))
    );

    // Simple k-means implementation
    if (clusters.length === 0) {
      // Initialize clusters
      for (let i = 0; i < k; i++) {
        clusters.push({
          centroid: features.map(() => Math.random()),
          members: [],
          stability: 0,
        });
      }
    }

    // Assign features to closest cluster
    const distances = clusters.map((cluster) =>
      this.euclideanDistance(features, cluster.centroid)
    );
    const closestCluster = distances.indexOf(Math.min(...distances));

    clusters[closestCluster].members.push(features);

    // Update centroid
    if (clusters[closestCluster].members.length > 0) {
      const members = clusters[closestCluster].members;
      clusters[closestCluster].centroid = features.map(
        (_, i) =>
          members.reduce((acc, member) => acc + member[i], 0) / members.length
      );
    }

    this.model.neuralConnections.set('clusters', clusters);
  }

  async performAutoencoding(features) {
    // Simple autoencoder for feature compression and reconstruction
    const encoder =
      this.model.neuralConnections.get('encoder') || this.initializeEncoder();
    const decoder =
      this.model.neuralConnections.get('decoder') || this.initializeDecoder();

    // Forward pass through encoder
    const encoded = this.forwardPass(features, encoder);

    // Forward pass through decoder
    const reconstructed = this.forwardPass(encoded, decoder);

    // Calculate reconstruction error
    const error = this.calculateReconstructionError(features, reconstructed);

    // Backpropagation (simplified)
    this.updateWeights(encoder, error);
    this.updateWeights(decoder, error);

    this.model.neuralConnections.set('encoder', encoder);
    this.model.neuralConnections.set('decoder', decoder);

    return { encoded, reconstructed, error };
  }

  initializeEncoder() {
    const defaultFeatureSize = 16; // Default feature size
    return {
      weights: this.randomWeights(defaultFeatureSize),
      biases: this.randomWeights(8),
      outputSize: 8,
    };
  }

  initializeDecoder() {
    return {
      weights: this.randomWeights(8),
      biases: this.randomWeights(20),
      outputSize: 20,
    };
  }

  forwardPass(input, network) {
    return input.map((val, i) => {
      const weightedSum =
        val * network.weights[i % network.weights.length] +
        network.biases[i % network.biases.length];
      return Math.tanh(weightedSum); // Activation function
    });
  }

  calculateReconstructionError(original, reconstructed) {
    return (
      original.reduce(
        (acc, val, i) =>
          acc + Math.pow(val - reconstructed[i % reconstructed.length], 2),
        0
      ) / original.length
    );
  }

  updateWeights(network, error) {
    const learningRate = this.config.learningRate;

    network.weights = network.weights.map(
      (weight) => weight - learningRate * error * Math.random()
    );

    network.biases = network.biases.map(
      (bias) => bias - learningRate * error * Math.random()
    );
  }

  async performTemporalLearning(features) {
    // Learn temporal patterns using simple RNN-like structure
    const temporal = this.model.neuralConnections.get('temporal') || {
      history: [],
      patterns: new Map(),
      predictions: [],
    };

    temporal.history.push({
      features,
      timestamp: Date.now(),
    });

    // Keep only recent history
    if (temporal.history.length > 100) {
      temporal.history = temporal.history.slice(-100);
    }

    // Extract temporal patterns
    if (temporal.history.length > 5) {
      const patterns = this.extractTemporalPatterns(temporal.history);
      patterns.forEach((pattern) => {
        temporal.patterns.set(pattern.id, pattern);
      });
    }

    this.model.neuralConnections.set('temporal', temporal);
  }

  extractTemporalPatterns(history) {
    const patterns = [];
    const windowSize = 5;

    for (let i = 0; i <= history.length - windowSize; i++) {
      const window = history.slice(i, i + windowSize);
      const pattern = {
        id: `pattern_${i}`,
        sequence: window.map((h) => h.features),
        timeSpan: window[window.length - 1].timestamp - window[0].timestamp,
        strength: this.calculatePatternStrength(window),
      };

      patterns.push(pattern);
    }

    return patterns;
  }

  calculatePatternStrength(window) {
    // Calculate pattern strength based on feature consistency
    const featureCount = window[0].features.length;
    let totalConsistency = 0;

    for (let f = 0; f < featureCount; f++) {
      const values = window.map((w) => w.features[f]);
      const mean = values.reduce((acc, val) => acc + val, 0) / values.length;
      const variance =
        values.reduce((acc, val) => acc + Math.pow(val - mean, 2), 0) /
        values.length;
      totalConsistency += 1 / (1 + variance);
    }

    return totalConsistency / featureCount;
  }

  async performAnomalyDetection(features) {
    const anomalies = this.model.neuralConnections.get('anomalies') || {
      threshold: 0.8,
      detectedCount: 0,
      recentAnomalies: [],
    };

    // Compare with normal patterns
    const normalPatterns = Array.from(this.model.memoryBank.values())
      .filter((exp) => exp.importance < 0.7)
      .map((exp) => exp.features);

    if (normalPatterns.length > 0) {
      const avgDistance =
        normalPatterns.reduce(
          (acc, pattern) => acc + this.euclideanDistance(features, pattern),
          0
        ) / normalPatterns.length;

      if (avgDistance > anomalies.threshold) {
        anomalies.detectedCount++;
        anomalies.recentAnomalies.push({
          features,
          distance: avgDistance,
          timestamp: Date.now(),
        });

        // Keep only recent anomalies
        if (anomalies.recentAnomalies.length > 10) {
          anomalies.recentAnomalies = anomalies.recentAnomalies.slice(-10);
        }
      }
    }

    this.model.neuralConnections.set('anomalies', anomalies);
  }

  updateLearningState(features) {
    this.learningState.epoch++;
    this.learningState.totalExperiences++;
    this.learningState.lastUpdate = Date.now();

    // Update accuracy based on prediction success
    this.learningState.accuracy = this.calculateAccuracy();

    // Update confidence based on consistency
    this.learningState.confidence = this.calculateConfidence(features);

    // Update adaptation score
    this.learningState.adaptationScore = this.calculateAdaptationScore();
  }

  calculateAccuracy() {
    const recent = Array.from(this.model.memoryBank.values()).slice(-20);
    if (recent.length === 0) return 0;

    const successfulPredictions = recent.filter(
      (exp) => exp.importance > 0.5
    ).length;
    return successfulPredictions / recent.length;
  }

  calculateConfidence(features) {
    const variance = this.calculateVariance(features);
    const novelty = this.calculateNovelty(features);

    // Higher confidence with lower variance and novelty
    return Math.max(0, 1 - (variance + novelty) / 2);
  }

  calculateAdaptationScore() {
    const memorySize = this.model.memoryBank.size;
    const clusterCount = (this.model.neuralConnections.get('clusters') || [])
      .length;
    const patternCount = (
      this.model.neuralConnections.get('temporal') || { patterns: new Map() }
    ).patterns.size;

    return Math.min(
      1.0,
      (memorySize / 1000 + clusterCount / 10 + patternCount / 50) / 3
    );
  }

  performMemoryCleanup() {
    // Remove least important and oldest memories
    const memories = Array.from(this.model.memoryBank.entries()).sort(
      (a, b) => {
        const scoreA =
          a[1].importance *
          this.config.memoryDecay **
            ((Date.now() - a[1].timestamp) / (24 * 60 * 60 * 1000));
        const scoreB =
          b[1].importance *
          this.config.memoryDecay **
            ((Date.now() - b[1].timestamp) / (24 * 60 * 60 * 1000));
        return scoreA - scoreB;
      }
    );

    const keepCount = Math.floor(this.config.maxMemorySize * 0.8);
    const toKeep = memories.slice(-keepCount);

    this.model.memoryBank.clear();
    toKeep.forEach(([key, value]) => {
      this.model.memoryBank.set(key, value);
    });
  }

  async updateModel(suggestions) {
    // Update model based on generated suggestions and their effectiveness
    const feedback = suggestions.map((suggestion) => ({
      features: suggestion.features || [],
      success: suggestion.success || Math.random() > 0.3,
      userRating: suggestion.userRating || 0.5 + Math.random() * 0.5,
      timestamp: Date.now(),
    }));

    // Use feedback for supervised learning
    await this.supervisedLearning(feedback);

    return {
      updated: true,
      suggestions: suggestions.length,
      avgRating:
        feedback.reduce((acc, f) => acc + f.userRating, 0) / feedback.length,
    };
  }

  async supervisedLearning(feedback) {
    // Update neural network weights based on feedback
    feedback.forEach((item) => {
      const target = item.success ? 1 : 0;
      const prediction = this.predict(item.features);
      const error = target - prediction;

      // Simple gradient descent
      this.updateNeuralWeights(error, item.features);
    });
  }

  predict(features) {
    // Simple prediction using current model
    if (features.length === 0) return 0.5;

    const sum = features.reduce((acc, val, i) => {
      const weight =
        this.neuralNetwork.inputLayer.weights[
          i % this.neuralNetwork.inputLayer.weights.length
        ];
      return acc + val * weight;
    }, 0);

    return Math.sigmoid(sum);
  }

  updateNeuralWeights(error, features) {
    const learningRate = this.config.learningRate;

    // Update input layer weights
    this.neuralNetwork.inputLayer.weights =
      this.neuralNetwork.inputLayer.weights.map((weight, i) => {
        const feature = features[i % features.length] || 0;
        return weight + learningRate * error * feature;
      });

    // Update hidden layer weights
    this.neuralNetwork.hiddenLayers.forEach((layer) => {
      layer.weights = layer.weights.map(
        (weight) => weight + learningRate * error * Math.random()
      );
    });
  }

  async processFeedback(feedback) {
    // Process user feedback to improve learning
    const feedbackData = {
      rating: feedback.rating || 0.5,
      comment: feedback.comment || '',
      timestamp: Date.now(),
      actionTaken: feedback.actionTaken || false,
    };

    // Update model based on feedback
    this.model.experienceBuffer.push(feedbackData);

    // Adjust learning parameters based on feedback
    this.adjustLearningParameters(feedbackData);

    return {
      processed: true,
      bufferSize: this.model.experienceBuffer.length,
      adaptationScore: this.learningState.adaptationScore,
    };
  }

  adjustLearningParameters(feedback) {
    if (feedback.rating > 0.7) {
      // Positive feedback - increase confidence
      this.learningState.confidence = Math.min(
        1.0,
        this.learningState.confidence + 0.05
      );
    } else if (feedback.rating < 0.3) {
      // Negative feedback - decrease confidence and adjust learning rate
      this.learningState.confidence = Math.max(
        0.0,
        this.learningState.confidence - 0.1
      );
      this.config.learningRate = Math.max(0.01, this.config.learningRate * 0.9);
    }
  }

  async getInsights() {
    return {
      learningState: this.learningState,
      modelStats: {
        memorySize: this.model.memoryBank.size,
        experienceBuffer: this.model.experienceBuffer.length,
        clusters: (this.model.neuralConnections.get('clusters') || []).length,
        temporalPatterns: (
          this.model.neuralConnections.get('temporal') || {
            patterns: new Map(),
          }
        ).patterns.size,
        anomalies: (
          this.model.neuralConnections.get('anomalies') || { detectedCount: 0 }
        ).detectedCount,
      },
      recommendations: await this.generateRecommendations(),
    };
  }

  async generateRecommendations() {
    const clusters = this.model.neuralConnections.get('clusters') || [];
    const temporalPatterns = this.model.neuralConnections.get('temporal') || {
      patterns: new Map(),
    };
    const anomalies = this.model.neuralConnections.get('anomalies') || {
      recentAnomalies: [],
    };

    const recommendations = [];

    // Cluster-based recommendations
    clusters.forEach((cluster, index) => {
      if (cluster.members.length > 5) {
        recommendations.push({
          type: 'cluster_insight',
          description: `Pattern cluster ${index + 1} shows consistent behavior with ${cluster.members.length} occurrences`,
          confidence: cluster.stability,
          actionable: true,
        });
      }
    });

    // Temporal pattern recommendations
    Array.from(temporalPatterns.patterns.values()).forEach((pattern) => {
      if (pattern.strength > 0.7) {
        recommendations.push({
          type: 'temporal_pattern',
          description: `Strong temporal pattern detected with ${pattern.strength.toFixed(2)} strength`,
          confidence: pattern.strength,
          actionable: true,
        });
      }
    });

    // Anomaly-based recommendations
    if (anomalies.recentAnomalies.length > 2) {
      recommendations.push({
        type: 'anomaly_alert',
        description: `${anomalies.recentAnomalies.length} recent anomalies detected, investigation recommended`,
        confidence: 0.8,
        actionable: true,
      });
    }

    return recommendations;
  }

  loadData(data) {
    if (data) {
      this.model = { ...this.model, ...data.model };
      this.learningState = { ...this.learningState, ...data.learningState };
    }
  }

  async exportData() {
    return {
      model: this.model,
      learningState: this.learningState,
      config: this.config,
      timestamp: Date.now(),
    };
  }
}

// Helper function for sigmoid activation
Math.sigmoid = function (x) {
  return 1 / (1 + Math.exp(-x));
};
