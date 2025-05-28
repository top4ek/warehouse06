# spec/models/node_spec.rb
require 'spec_helper' # Loads RSpec configuration, RACK_ENV='test', and the app

RSpec.describe Node, type: :model do
  describe '#type' do
    context 'when the node has a digest (is a file)' do
      let(:file_node) { Node.new(digest: 'somesha256digest') }

      it 'returns "file"' do
        expect(file_node.type).to eq('file')
      end
    end

    context 'when the node does not have a digest (is a directory)' do
      let(:directory_node) { Node.new(digest: nil) } # Or just Node.new if digest is nil by default

      it 'returns "directory"' do
        expect(directory_node.type).to eq('directory')
      end
    end
  end

  # Add more tests for Node model later, e.g. validations if any are added.
  # For example, if path was required:
  # describe 'validations' do
  #   it 'is invalid without a path' do
  #     node = Node.new(path: nil)
  #     # This requires a DB connection and sequel validation helpers configured
  #     # expect(node.valid?).to be_falsey 
  #     # expect(node.errors[:path]).to include("is not present")
  #   end
  # end
end
