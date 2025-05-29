# frozen_string_literal: true

require 'spec_helper'

RSpec.describe Node, type: :model do
  describe '#type' do
    context 'when the node has a digest (is a file)' do
      let(:file_node) { Node.new(digest: 'somesha256digest') }

      it 'returns "file"' do
        expect(file_node.type).to eq('file')
      end
    end

    context 'when the node does not have a digest (is a directory)' do
      let(:directory_node) { Node.new(digest: nil) }

      it 'returns "directory"' do
        expect(directory_node.type).to eq('directory')
      end
    end
  end
end
