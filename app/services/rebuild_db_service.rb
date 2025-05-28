# frozen_string_literal: true

require 'find'
require_relative 'rebuild_db_service/node_creator'

class RebuildDbService
  STORAGE_ROOT = 'storage'

  def initialize(root: STORAGE_ROOT)
    @data_root = Pathname(Dir.pwd).join(root)
  end

  def call
    processed_count = 0
    Find.find(@data_root).each do |f|
      Pathname.new(f)
              .relative_path_from(@data_root)
              .descend do |it|
                NodeCreator.new(@data_root.join(it)).call
                processed_count += 1
              end
    end
    processed_count
  end

  private

  def clean_db
    puts('Cleaning database.')
    Tagging.dataset.delete
    Tag.dataset.delete
    Node.dataset.delete # Changed from Document.dataset.delete
  end
end
