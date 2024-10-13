require 'digest'
require 'find'
require 'front_matter_parser'

class RebuildDbService
  def initialize(root: 'storage')
    @data_root = Pathname(Dir.pwd).join(root)
    @files = Find.find(@data_root)
                 .map { |i| Pathname(i) }.reject(&:directory?)
                 .map { |i| { file: i, path: i.relative_path_from(@data_root).to_s.split('/') } }
  end

  def call
    clean_db
    @files.each do |f|
      size = f[:path].size - 1
      parent = Directory.find_or_create(parent_id: nil, key: 'root')
      f[:path].each_with_index do |key, idx|
        if idx == size
          if key == 'index.md'
            data = FrontMatterParser::Parser.parse_file(f[:file])
            parent.update(name: data.front_matter['name'], description: data.content)
          else
            data = File.read(f[:file])
            Document.create(directory_id: parent&.id,
                            digest: Digest::SHA256.hexdigest(data),
                            size: data.size,
                            path: f[:file].to_s,
                            key:)

          end
        else
          parent = Directory.find_or_create(parent_id: parent&.id, key:)
        end
      end
    end
  end

  private

  def fetch_repo; end

  def parse_md(path)
    FrontMatterParser::Parser.parse_file(path)
  end

  def clean_db
    puts('Clean database.')
    Tagging.dataset.delete
    Tag.dataset.delete
    Document.dataset.delete
    Directory.dataset.delete
  end
end
