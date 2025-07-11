#!/usr/bin/env python3
"""
ZSH Clipboard Image Handler - Autonomous Learning Seed Project

This is the first seed utility spawned by KaskMan's autonomous learning system.
It detects clipboard image content and replaces it with a filepath after storing
the image in a determined directory.

Evolution Plan:
- Seed Phase: Basic clipboard -> file functionality
- Growth Phase: Learn user patterns, optimize storage
- Expansion Phase: Support multiple formats, cloud integration
- Autonomous Phase: Self-improve based on usage analytics

Part of the KaskMan Autonomous R&D System
"""

import os
import sys
import io
import hashlib
import json
import time
import subprocess
from pathlib import Path
from datetime import datetime
from typing import Optional, Dict, Any, List
import tempfile
import argparse

try:
    from PIL import Image, ImageGrab
    PIL_AVAILABLE = True
except ImportError:
    PIL_AVAILABLE = False

class ClipboardImageHandler:
    """
    Autonomous clipboard image handler with learning capabilities.
    
    This class represents the seed implementation that will evolve
    through the autonomous learning system.
    """
    
    def __init__(self, config_dir: str = None):
        self.config_dir = config_dir or os.path.expanduser("~/.kaskman/clipboard")
        self.storage_dir = os.path.join(self.config_dir, "images")
        self.config_file = os.path.join(self.config_dir, "config.json")
        self.learning_file = os.path.join(self.config_dir, "learning_data.json")
        
        # Initialize directories
        os.makedirs(self.config_dir, exist_ok=True)
        os.makedirs(self.storage_dir, exist_ok=True)
        
        # Load configuration and learning data
        self.config = self.load_config()
        self.learning_data = self.load_learning_data()
        
        # Initialize learning metrics
        self.session_start = time.time()
        self.usage_count = 0
        
    def load_config(self) -> Dict[str, Any]:
        """Load configuration with autonomous learning defaults."""
        default_config = {
            "storage_pattern": "{date}/{hash}_{timestamp}.{ext}",
            "supported_formats": ["png", "jpg", "jpeg", "gif", "bmp", "tiff"],
            "max_file_size": 10 * 1024 * 1024,  # 10MB
            "learning_enabled": True,
            "auto_optimize": True,
            "compression_quality": 85,
            "naming_strategy": "hash_timestamp",
            
            # Autonomous learning parameters
            "learning_threshold": 10,  # Start learning after 10 uses
            "pattern_recognition": True,
            "auto_expansion": True,
            "evolution_triggers": {
                "usage_frequency": 50,
                "user_satisfaction": 0.8,
                "performance_threshold": 0.9
            }
        }
        
        if os.path.exists(self.config_file):
            try:
                with open(self.config_file, 'r') as f:
                    user_config = json.load(f)
                default_config.update(user_config)
            except Exception as e:
                print(f"Warning: Could not load config: {e}", file=sys.stderr)
        
        return default_config
    
    def load_learning_data(self) -> Dict[str, Any]:
        """Load learning data for autonomous improvement."""
        default_learning = {
            "usage_patterns": {},
            "user_preferences": {},
            "performance_metrics": {
                "total_uses": 0,
                "success_rate": 1.0,
                "average_processing_time": 0.0,
                "user_satisfaction": 0.8,
                "error_count": 0
            },
            "evolution_history": [],
            "friction_points": [],
            "improvement_suggestions": []
        }
        
        if os.path.exists(self.learning_file):
            try:
                with open(self.learning_file, 'r') as f:
                    learning_data = json.load(f)
                default_learning.update(learning_data)
            except Exception as e:
                print(f"Warning: Could not load learning data: {e}", file=sys.stderr)
        
        return default_learning
    
    def save_learning_data(self):
        """Save learning data for autonomous evolution."""
        try:
            with open(self.learning_file, 'w') as f:
                json.dump(self.learning_data, f, indent=2, default=str)
        except Exception as e:
            print(f"Warning: Could not save learning data: {e}", file=sys.stderr)
    
    def detect_clipboard_image(self) -> Optional[Image.Image]:
        """
        Detect if clipboard contains image data.
        
        Returns:
            PIL Image object if image found, None otherwise
        """
        start_time = time.time()
        
        try:
            if PIL_AVAILABLE:
                # Try PIL ImageGrab first (works on macOS and Windows)
                image = ImageGrab.grabclipboard()
                if image and isinstance(image, Image.Image):
                    self.record_performance_metric("detection_time", time.time() - start_time)
                    return image
            
            # Fallback to platform-specific clipboard access
            if sys.platform == "darwin":  # macOS
                return self._detect_macos_clipboard_image()
            elif sys.platform.startswith("linux"):
                return self._detect_linux_clipboard_image()
            elif sys.platform == "win32":
                return self._detect_windows_clipboard_image()
                
        except Exception as e:
            self.record_error("clipboard_detection", str(e))
            
        return None
    
    def _detect_macos_clipboard_image(self) -> Optional[Image.Image]:
        """Detect clipboard image on macOS using pbpaste."""
        try:
            # Check if clipboard contains image data
            result = subprocess.run(
                ["pbpaste", "-Prefer", "png"],
                capture_output=True,
                check=False
            )
            
            if result.returncode == 0 and result.stdout:
                image = Image.open(io.BytesIO(result.stdout))
                return image
                
        except Exception as e:
            self.record_error("macos_clipboard", str(e))
        
        return None
    
    def _detect_linux_clipboard_image(self) -> Optional[Image.Image]:
        """Detect clipboard image on Linux using xclip."""
        try:
            # Try to get image from clipboard
            result = subprocess.run(
                ["xclip", "-selection", "clipboard", "-t", "image/png", "-o"],
                capture_output=True,
                check=False
            )
            
            if result.returncode == 0 and result.stdout:
                image = Image.open(io.BytesIO(result.stdout))
                return image
                
        except Exception as e:
            self.record_error("linux_clipboard", str(e))
        
        return None
    
    def _detect_windows_clipboard_image(self) -> Optional[Image.Image]:
        """Detect clipboard image on Windows."""
        try:
            if PIL_AVAILABLE:
                return ImageGrab.grabclipboard()
        except Exception as e:
            self.record_error("windows_clipboard", str(e))
        
        return None
    
    def generate_filename(self, image: Image.Image, extension: str = "png") -> str:
        """
        Generate filename using configured strategy with learning.
        
        Args:
            image: PIL Image object
            extension: File extension
            
        Returns:
            Generated filename
        """
        timestamp = datetime.now()
        
        # Generate hash of image content for deduplication
        img_bytes = io.BytesIO()
        image.save(img_bytes, format='PNG')
        content_hash = hashlib.md5(img_bytes.getvalue()).hexdigest()[:8]
        
        # Learn from user preferences
        if self.config["naming_strategy"] == "hash_timestamp":
            filename = f"{content_hash}_{int(time.time())}.{extension}"
        elif self.config["naming_strategy"] == "descriptive":
            # Future evolution: Use AI to generate descriptive names
            filename = f"image_{content_hash}_{timestamp.strftime('%Y%m%d_%H%M%S')}.{extension}"
        else:
            filename = f"clipboard_{int(time.time())}.{extension}"
        
        # Apply storage pattern
        pattern = self.config["storage_pattern"]
        formatted_path = pattern.format(
            date=timestamp.strftime("%Y-%m-%d"),
            hash=content_hash,
            timestamp=int(time.time()),
            ext=extension
        )
        
        return formatted_path
    
    def store_image(self, image: Image.Image) -> str:
        """
        Store image in determined directory and return filepath.
        
        Args:
            image: PIL Image object
            
        Returns:
            Full filepath to stored image
        """
        start_time = time.time()
        
        try:
            # Generate filename
            filename = self.generate_filename(image)
            full_path = os.path.join(self.storage_dir, filename)
            
            # Ensure directory exists
            os.makedirs(os.path.dirname(full_path), exist_ok=True)
            
            # Optimize image if configured
            if self.config["auto_optimize"]:
                image = self.optimize_image(image)
            
            # Save image
            image.save(full_path, optimize=True, quality=self.config["compression_quality"])
            
            # Record learning data
            self.record_storage_success(full_path, time.time() - start_time)
            
            return full_path
            
        except Exception as e:
            self.record_error("image_storage", str(e))
            raise
    
    def optimize_image(self, image: Image.Image) -> Image.Image:
        """
        Optimize image based on learned preferences.
        
        Args:
            image: Original PIL Image
            
        Returns:
            Optimized PIL Image
        """
        # Learn optimal dimensions based on usage
        width, height = image.size
        max_dimension = self.learning_data["user_preferences"].get("max_dimension", 2048)
        
        if width > max_dimension or height > max_dimension:
            # Maintain aspect ratio
            ratio = min(max_dimension / width, max_dimension / height)
            new_size = (int(width * ratio), int(height * ratio))
            image = image.resize(new_size, Image.Resampling.LANCZOS)
        
        return image
    
    def replace_clipboard_with_path(self, filepath: str) -> bool:
        """
        Replace clipboard content with the file path.
        
        Args:
            filepath: Path to the stored image
            
        Returns:
            True if successful, False otherwise
        """
        try:
            if sys.platform == "darwin":  # macOS
                subprocess.run(["pbcopy"], input=filepath.encode(), check=True)
            elif sys.platform.startswith("linux"):
                subprocess.run(["xclip", "-selection", "clipboard"], input=filepath.encode(), check=True)
            elif sys.platform == "win32":
                # Windows clipboard handling
                subprocess.run(["clip"], input=filepath.encode(), shell=True, check=True)
            
            self.record_replacement_success()
            return True
            
        except Exception as e:
            self.record_error("clipboard_replacement", str(e))
            return False
    
    def process_clipboard(self) -> Optional[str]:
        """
        Main processing function: detect, store, and replace clipboard image.
        
        Returns:
            Filepath if successful, None if no image or error
        """
        start_time = time.time()
        self.usage_count += 1
        
        try:
            # Detect clipboard image
            image = self.detect_clipboard_image()
            if not image:
                return None
            
            # Store image
            filepath = self.store_image(image)
            
            # Replace clipboard with path
            if self.replace_clipboard_with_path(filepath):
                # Record success metrics
                processing_time = time.time() - start_time
                self.record_processing_success(filepath, processing_time)
                
                # Check for evolution triggers
                self.check_evolution_triggers()
                
                return filepath
            
        except Exception as e:
            self.record_error("process_clipboard", str(e))
        
        return None
    
    def record_processing_success(self, filepath: str, processing_time: float):
        """Record successful processing for learning."""
        metrics = self.learning_data["performance_metrics"]
        metrics["total_uses"] += 1
        
        # Update average processing time
        total_time = metrics["average_processing_time"] * (metrics["total_uses"] - 1)
        metrics["average_processing_time"] = (total_time + processing_time) / metrics["total_uses"]
        
        # Update success rate
        total_attempts = metrics["total_uses"] + metrics["error_count"]
        metrics["success_rate"] = metrics["total_uses"] / total_attempts
        
        # Record usage pattern
        hour = datetime.now().hour
        day_of_week = datetime.now().weekday()
        
        patterns = self.learning_data["usage_patterns"]
        patterns[f"hour_{hour}"] = patterns.get(f"hour_{hour}", 0) + 1
        patterns[f"day_{day_of_week}"] = patterns.get(f"day_{day_of_week}", 0) + 1
        
        self.save_learning_data()
    
    def record_error(self, error_type: str, error_message: str):
        """Record errors for learning and improvement."""
        self.learning_data["performance_metrics"]["error_count"] += 1
        
        error_entry = {
            "timestamp": time.time(),
            "type": error_type,
            "message": error_message
        }
        
        if "errors" not in self.learning_data:
            self.learning_data["errors"] = []
        
        self.learning_data["errors"].append(error_entry)
        
        # Keep only recent errors
        if len(self.learning_data["errors"]) > 100:
            self.learning_data["errors"] = self.learning_data["errors"][-50:]
        
        self.save_learning_data()
    
    def record_performance_metric(self, metric_name: str, value: float):
        """Record performance metrics for optimization."""
        if "detailed_metrics" not in self.learning_data:
            self.learning_data["detailed_metrics"] = {}
        
        metrics = self.learning_data["detailed_metrics"]
        if metric_name not in metrics:
            metrics[metric_name] = []
        
        metrics[metric_name].append({
            "timestamp": time.time(),
            "value": value
        })
        
        # Keep only recent metrics
        if len(metrics[metric_name]) > 1000:
            metrics[metric_name] = metrics[metric_name][-500:]
    
    def record_storage_success(self, filepath: str, storage_time: float):
        """Record successful storage operation."""
        self.record_performance_metric("storage_time", storage_time)
        
        # Learn about file sizes and formats
        try:
            file_size = os.path.getsize(filepath)
            self.record_performance_metric("file_size", file_size)
        except Exception:
            pass
    
    def record_replacement_success(self):
        """Record successful clipboard replacement."""
        self.record_performance_metric("replacement_success", 1)
    
    def check_evolution_triggers(self):
        """Check if conditions are met for autonomous evolution."""
        triggers = self.config["evolution_triggers"]
        metrics = self.learning_data["performance_metrics"]
        
        # Check usage frequency trigger
        if metrics["total_uses"] >= triggers["usage_frequency"]:
            self.trigger_evolution("usage_frequency")
        
        # Check user satisfaction trigger
        if metrics["success_rate"] >= triggers["performance_threshold"]:
            self.trigger_evolution("performance_excellence")
    
    def trigger_evolution(self, trigger_reason: str):
        """Trigger autonomous evolution to next phase."""
        evolution_entry = {
            "timestamp": time.time(),
            "trigger": trigger_reason,
            "phase_transition": "seed_to_growth",
            "metrics_snapshot": self.learning_data["performance_metrics"].copy()
        }
        
        self.learning_data["evolution_history"].append(evolution_entry)
        
        # Signal to KaskMan system for evolution
        evolution_signal_file = os.path.join(self.config_dir, "evolution_ready.json")
        with open(evolution_signal_file, 'w') as f:
            json.dump(evolution_entry, f, indent=2, default=str)
        
        print(f"Evolution triggered: {trigger_reason}", file=sys.stderr)
    
    def get_learning_insights(self) -> Dict[str, Any]:
        """Get insights from learning data."""
        metrics = self.learning_data["performance_metrics"]
        patterns = self.learning_data["usage_patterns"]
        
        # Identify peak usage hours
        peak_hour = max(patterns.keys(), key=lambda k: patterns[k] if k.startswith("hour_") else 0)
        peak_day = max(patterns.keys(), key=lambda k: patterns[k] if k.startswith("day_") else 0)
        
        return {
            "performance": {
                "total_uses": metrics["total_uses"],
                "success_rate": f"{metrics['success_rate']:.2%}",
                "avg_processing_time": f"{metrics['average_processing_time']:.3f}s",
                "error_count": metrics["error_count"]
            },
            "usage_patterns": {
                "peak_hour": peak_hour.replace("hour_", "") if peak_hour.startswith("hour_") else "unknown",
                "peak_day": peak_day.replace("day_", "") if peak_day.startswith("day_") else "unknown"
            },
            "evolution_status": {
                "evolution_count": len(self.learning_data["evolution_history"]),
                "ready_for_next_phase": metrics["total_uses"] >= self.config["evolution_triggers"]["usage_frequency"]
            }
        }


def main():
    """Main entry point for the clipboard handler."""
    parser = argparse.ArgumentParser(description="ZSH Clipboard Image Handler - Autonomous Learning Seed")
    parser.add_argument("--process", action="store_true", help="Process clipboard image")
    parser.add_argument("--status", action="store_true", help="Show learning status")
    parser.add_argument("--config-dir", help="Configuration directory")
    parser.add_argument("--verbose", "-v", action="store_true", help="Verbose output")
    
    args = parser.parse_args()
    
    # Initialize handler
    handler = ClipboardImageHandler(config_dir=args.config_dir)
    
    if args.status:
        # Show learning insights
        insights = handler.get_learning_insights()
        print(json.dumps(insights, indent=2))
        return
    
    if args.process or len(sys.argv) == 1:
        # Process clipboard
        result = handler.process_clipboard()
        if result:
            if args.verbose:
                print(f"Image stored: {result}", file=sys.stderr)
            print(result)  # Output filepath for ZSH integration
        else:
            if args.verbose:
                print("No image found in clipboard", file=sys.stderr)
            sys.exit(1)


if __name__ == "__main__":
    main()