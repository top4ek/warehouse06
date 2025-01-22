# frozen_string_literal: true

require 'find'
require_relative 'rebuild_db_service/node_creator'

class RebuildDbService
  STORAGE_ROOT = 'storage'

  def initialize(root: STORAGE_ROOT)
    @data_root = Pathname(Dir.pwd).join(root)
  end

  def call
    Find.find(@data_root).each do |f|
      Pathname.new(f)
              .relative_path_from(@data_root)
              .descend { NodeCreator.new(@data_root.join(it)).call }
    end
  end

  private

  def clean_db
    puts('Cleaning database.')
    Tagging.dataset.delete
    Tag.dataset.delete
    Document.dataset.delete
  end
end
