# frozen_string_literal: true

require 'redcarpet'

# CustomHtmlRenderer is now expected to be autoloaded from lib/renderers/
# or explicitly required if not using an autoloader that covers lib/.

module Nodes
  module Show
    class << self
      include Dry::Monads[:result]

      def call(request)
        node_record = document(request.path)

        if node_record.nil?
          # Ensure the failure response matches what the application expects for a 404
          return Failure({ status: 404, body: 'Page not found.', title: 'Not Found', file_to_send: false, path: request.path })
        end

        # Common success attributes
        result = { status: 200, title: node_record.name, path: node_record.path }

        if node_record.type == 'file'
          result[:file_to_send] = true # Indicate that a file should be sent
          result[:body] = node_record.full_storage_path # Path to the file on disk
        else # 'directory', implies markdown rendering
          result[:file_to_send] = false # Indicate that HTML body will be sent
          # Use CustomHtmlRenderer for markdown processing
          renderer = CustomHtmlRenderer.new(node_record.path, filter_html: true)
          markdown_parser = Redcarpet::Markdown.new(renderer, autolink: true, tables: true)
          result[:body] = markdown_parser.render(node_record.description || '') # Render description or empty string
        end
        Success(result.compact) # compact to remove any nil values if any
      end

      private

      def document(request_path)
        # Use Pathname to clean the path (handles ., .., //, etc.)
        # Start with a path relative to root, by removing any leading slash for Pathname.
        # If request_path is just "/", Pathname.new("") or Pathname.new(".") might result.
        # So, special handling for "/" or empty string first.

        if request_path == '/' || request_path.empty?
          # For root, directly look for the conventional root README.md node
          return Node.find(path: 'README.md')
        end

        # For non-root paths:
        # 1. Create a Pathname object. Prepending a dummy root like '/root' helps if paths are relative,
        #    but since request_path should be absolute (e.g. /foo/bar), we can use it directly.
        #    However, to match DB paths that are stored like "foo/bar" (no leading /),
        #    it's often easier to work with relative-to-root strings.
        #
        # Let's assume paths in DB are like: "README.md", "foo/bar.txt", "foo/baz/README.md"
        
        # Clean the path: remove leading slash, then clean.
        # Pathname("foo/bar/../baz").cleanpath -> Pathname("foo/baz")
        # Pathname("/foo/bar/").cleanpath.to_s -> "/foo/bar"
        
        cleaned_path_str = Pathname.new(request_path).cleanpath.to_s

        # Remove leading slash to match DB storage convention (e.g. "foo/bar" not "/foo/bar")
        # Also, if it was just "/", cleanpath makes it "/", so gsub removes it to ""
        # This means a request like "//" or "/." would also become "" after this.
        db_lookup_path = cleaned_path_str.gsub(%r{^/}, '')

        # If the cleaned path after removing leading / is empty, it implies it was some form of root.
        # This case should have been caught by `request_path == '/'` earlier, but as a safeguard:
        if db_lookup_path.empty? && request_path != '/' # e.g. request_path was "/." or "//"
            return Node.find(path: 'README.md')
        end
        # At this point, db_lookup_path is like "foo/bar" or "foo/file.txt"

        node = Node.find(path: db_lookup_path)

        if node
          return node
        else
          # If not found, try appending 'README.md'
          # File.join correctly handles adding README.md, even if db_lookup_path was for a file.
          # e.g. db_lookup_path = "foo/file.txt", readme_path becomes "foo/file.txt/README.md" (unlikely to exist)
          # e.g. db_lookup_path = "foo/bar", readme_path becomes "foo/bar/README.md" (what we want to check)
          readme_path = File.join(db_lookup_path, 'README.md')
          node_readme = Node.find(path: readme_path)
          return node_readme if node_readme&.type == 'directory'
        end

        nil
      end
    end
  end
end
