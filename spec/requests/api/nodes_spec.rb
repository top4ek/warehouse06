# spec/requests/api/nodes_spec.rb
require 'spec_helper' # Loads RSpec, Rack::Test, and the app via config.ru

RSpec.describe 'GET /api/nodes', type: :request do
  # `app` method is defined in spec_helper.rb and returns the Roda Application class

  context 'when there are no nodes' do
    before do
      # Ensure DB is clean for this test context if needed.
      # For now, assumes RebuildDbService might run or DB is clean.
      # A more robust setup would ensure a clean state.
      Node.dataset.delete # Clear any existing nodes
    end

    it 'returns a 200 OK status' do
      get '/api/nodes'
      expect(last_response.status).to eq(200)
    end

    it 'returns an empty JSON array' do
      get '/api/nodes'
      expect(last_response.content_type).to eq('application/json')
      expect(JSON.parse(last_response.body)).to eq([])
    end
  end

  context 'when there are some nodes' do
    before do
      Node.dataset.delete # Clear any existing nodes
      create(:node, name: 'File 1', path: 'file1.txt') # Default factory is a file
      create(:node, :directory, name: 'Directory 1', path: 'dir1')
    end

    it 'returns a 200 OK status' do
      get '/api/nodes'
      expect(last_response.status).to eq(200)
    end

    it 'returns the nodes as JSON' do
      get '/api/nodes'
      expect(last_response.content_type).to eq('application/json')
      parsed_body = JSON.parse(last_response.body)
      expect(parsed_body.size).to eq(2)
      # Order might not be guaranteed, so check for names present
      expect(parsed_body.map{ |n| n['name'] }).to match_array(['File 1', 'Directory 1'])
    end
  end
end
