# frozen_string_literal: true

require 'spec_helper'
require 'fileutils' # For creating dummy files for send_file tests

RSpec.describe 'Web Routes', type: :request do
  include Rack::Test::Methods # Required for last_response, get, etc.

  # Helper to define the app for Rack::Test
  # The instructions said Routes::Base, but the actual app is App.freeze.app or just App
  # Assuming App is the Roda application class that loads Routes::Base
  def app
    App.freeze.app
  end

  # Clean nodes before each test example for isolation
  before(:each) { Node.dataset.delete }

  # Clean up dummy files once after all tests in this spec file
  after(:all) do
    # Ensure the path is constructed safely and correctly
    cleanup_path = File.join(Dir.pwd, 'storage', 'test_files_for_specs')
    FileUtils.rm_rf(cleanup_path) if Dir.exist?(cleanup_path) # Only remove if it exists
  end

  describe 'GET / (Root Path)' do
    context 'when root README.md exists' do
      before do
        # digest: nil ensures it's treated as a directory/markdown content
        create(:node, path: 'README.md', name: 'Root README', description: '# Root Welcome', digest: nil)
      end

      it 'renders the root README.md' do
        get '/'
        expect(last_response.status).to eq(200)
        expect(last_response.body).to include('<h1>Root Welcome</h1>')
      end
    end
  end

  describe 'GET /some/directory/ (Directory Path)' do
    context 'when accessing a directory with a README.md' do
      before do
        create(:node, path: 'some/nested_dir/README.md', name: 'Nested README', description: '## Nested Content', digest: nil)
      end

      it 'renders the README.md when requested with a trailing slash' do
        get '/some/nested_dir/' # Request with trailing slash
        expect(last_response.status).to eq(200)
        expect(last_response.body).to include('<h2>Nested Content</h2>')
      end

      it 'renders the README.md when requested without a trailing slash (interactor normalizes)' do
        get '/some/nested_dir' # Request without trailing slash
        expect(last_response.status).to eq(200) # Interactor/Router handles path normalization
        expect(last_response.body).to include('<h2>Nested Content</h2>')
        # Optional: Check if redirection to canonical path happened if web.rb enforces it.
        # The current Nodes::Show interactor (Turn 20) normalizes the path for lookup,
        # and Web.rb (Turn 22) might redirect. If it redirects to /some/nested_dir/README.md,
        # last_request.url would reflect that.
        # If node.path is 'some/nested_dir/README.md', web.rb redirects if request.path is not /some/nested_dir/README.md
        # So, a request to '/some/nested_dir' for a node with path 'some/nested_dir/README.md' should redirect.
        expect(last_request.url).to match(%r{/some/nested_dir/README.md$})
      end
    end
  end

  describe 'GET /files/image.png (File Path)' do
    context 'when requesting a file' do
      let(:file_node_path) { 'test_files_for_specs/image.png' } # Relative path for the node
      # This is where the actual file will reside for 'send_file'
      let(:actual_file_path_on_disk) { File.join(Dir.pwd, 'storage', file_node_path) }

      before do
        FileUtils.mkdir_p(File.dirname(actual_file_path_on_disk))
        File.write(actual_file_path_on_disk, 'dummy image content')
        # The node's path should match what Nodes::Show expects for Node#full_storage_path
        create(:node, :file, path: file_node_path, name: 'image.png')
      end

      # No need for after hook here for this specific file, after(:all) cleans the top directory.

      it 'serves the file with correct headers and content' do
        get "/#{file_node_path}" # Request path matches the node's path

        expect(last_response.status).to eq(200)
        # Roda's send_file should infer Content-Type from file extension
        expect(last_response.headers['Content-Type']).to eq('image/png')
        # Content-Disposition should include the filename
        expect(last_response.headers['Content-Disposition']).to include('filename="image.png"')
        expect(last_response.body).to eq('dummy image content')
      end
    end
  end

  describe 'GET /non/existent/path (Not Found)' do
    context 'when requesting a non-existent path' do
      it 'returns a 404 status and error message' do
        get '/non/existent/path'
        expect(last_response.status).to eq(404)
        # Check for messages from the Failure payload in Nodes::Show (via Web.rb)
        expect(last_response.body).to include('Page not found.') # From payload[:body]
        expect(last_response.body).to include('Not Found')     # From payload[:title]
      end
    end
  end

  describe 'Image Rendering in Markdown (Verifying CustomHtmlRenderer)' do
    context 'when markdown contains a relative image path' do
      it 'correctly resolves relative image paths in rendered HTML' do
        # Case 1: Image in a subdirectory relative to the markdown file
        create(:node, path: 'articles/tech/README.md', name: 'Tech Article',
                      description: 'Check this: ![ схема ](diagram.svg)', digest: nil)
        # Optional: create(:node, :file, path: 'articles/tech/diagram.svg', name: 'diagram.svg')

        get '/articles/tech/README.md' # Request the markdown file directly
        expect(last_response.status).to eq(200)
        expect(last_response.body).to include('<img src="/articles/tech/diagram.svg" alt=" схема ">')

        # Case 2: Image in the same directory as the markdown file (root of a non-root dir)
        Node.dataset.delete # Clean for next sub-test
        create(:node, path: 'other_articles/README.md', name: 'Other Article',
                      description: 'Another: ![pic](photo.jpeg)', digest: nil)
        # Optional: create(:node, :file, path: 'other_articles/photo.jpeg', name: 'photo.jpeg')

        get '/other_articles/README.md' # Request the markdown file directly
        expect(last_response.status).to eq(200)
        expect(last_response.body).to include('<img src="/other_articles/photo.jpeg" alt="pic">')
      end
    end
  end
end
