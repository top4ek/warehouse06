# frozen_string_literal: true

require 'spec_helper'

RSpec.describe 'GET /api/nodes', type: :request do
  context 'when there are no nodes' do
    before do
      Node.dataset.delete
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
      Node.dataset.delete
      create(:node, name: 'File 1', path: 'file1.txt')
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
      expect(parsed_body.map{ |n| n['name'] }).to match_array(['File 1', 'Directory 1'])
    end
  end
end
