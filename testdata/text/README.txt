Ryan AI CLI - Test Data Files
=============================

This directory contains test data files for validating the file loading and embedding functionality of the Ryan AI CLI system.

File Structure:
--------------

code/
  - sample.go: Go source code with various constructs (interfaces, structs, methods)
  - sample.py: Python source code with async/await, classes, and type hints
  - sample.js: JavaScript/TypeScript file with modern ES6+ features
  - sample.rs: Rust source code with ownership patterns and error handling

text/
  - documentation.md: Comprehensive markdown documentation
  - README.txt: This plain text file with project information
  - notes.log: Log file format with timestamps and structured data
  - story.txt: Natural language text for semantic search testing

structured/
  - config.json: JSON configuration file
  - settings.yaml: YAML configuration file
  - database.sql: SQL schema and data
  - docker-compose.yml: Docker compose configuration

large/
  - big_document.txt: Large text file (>1MB) for chunking tests
  - large_codebase.go: Large source file for performance testing

corrupted/
  - invalid.txt: File with unusual encoding or binary data
  - empty.txt: Empty file for edge case testing
  - permissions.txt: File with restricted permissions

Usage in Tests:
--------------

These files are designed to test various aspects of the document indexing system:

1. File Type Detection: Different extensions should be properly categorized
2. Content Chunking: Large files should be split appropriately  
3. Metadata Extraction: File properties should be captured correctly
4. Semantic Search: Content should be searchable by meaning, not just keywords
5. Error Handling: Corrupted or inaccessible files should be handled gracefully
6. Performance: System should handle files of various sizes efficiently

Testing Scenarios:
-----------------

Basic Indexing:
- Index each file type and verify proper categorization
- Check that metadata includes file path, type, size, and modification time
- Verify content is properly extracted and chunked

Search Quality:
- Search for technical terms and verify relevant files are returned
- Test semantic search (e.g., "error handling" should find relevant code)
- Test cross-file relationships and dependencies

Performance:
- Measure indexing time for different file sizes
- Test concurrent indexing operations
- Verify memory usage stays within reasonable bounds

Edge Cases:
- Handle empty files gracefully
- Process files with unusual extensions
- Deal with binary or corrupted content appropriately

Integration:
- Test with real embedding providers (Ollama, OpenAI)
- Verify persistence across system restarts
- Test integration with chat memory system

Expected Behavior:
-----------------

After indexing these test files, the system should be able to:

1. Answer questions about the codebase structure and functionality
2. Find relevant files when asked about specific programming concepts
3. Provide context-aware responses based on file content
4. Handle various file formats and sizes efficiently
5. Maintain good performance even with larger datasets

Quality Metrics:
---------------

- Indexing Speed: All test files should index in <10 seconds
- Search Accuracy: Relevant files should appear in top 3 results
- Memory Usage: Peak memory should not exceed 200MB during testing
- Error Rate: <1% of operations should fail due to system issues

This test data provides comprehensive coverage for validating the robustness and effectiveness of Ryan's file loading and embedding capabilities.