# frozen_string_literal: true

require 'digest'
require 'front_matter_parser'

class NodeCreator
  attr_reader :path, :basename

  def initialize(path)
    @path = path.is_a?(Pathname) ? path : Pathname.new(path)
    @basename = @path.basename
  end

  def call
    return if basename.to_s == 'README.md' && !directory?

    set_content
    node.save
  end

  def directory?
    path.directory?
  end

  private

  def set_content
    if directory?
      if readme.nil?
        node.name = basename
      else
        node.description = readme.content
        node.name = readme.front_matter['name'] || basename
        node.date = readme_date
      end
    else
      node.digest = Digest::SHA256.hexdigest(raw_data)
      node.size = raw_data.size
    end
  end

  def readme_date
    Date.parse(readme.front_matter['date'].to_s)
  rescue StandardError
    nil
  end

  def raw_data
    return nil if directory?

    @raw_data ||= File.read(path)
  end

  def node
    return @node unless @node.nil?

    @node = Node.find_or_create(path: relative_path.to_s)
  end

  def data_root
    @data_root = Pathname(Dir.pwd).join(RebuildDbService::STORAGE_ROOT)
  end

  def relative_path
    @relative_path ||= path.relative_path_from(data_root)
  end

  def readme
    return @readme unless @readme.nil?

    full_path = path.join('README.md')
    return nil unless full_path.exist?

    @readme = FrontMatterParser::Parser.parse_file(full_path)
  end
end
