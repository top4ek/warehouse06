# frozen_string_literal: true

class Node < Sequel::Model
  plugin :validation_helpers

  raise_on_save_failure

  def type
    digest.nil? ? 'directory' : 'file'
  end

  def full_storage_path
    File.join(Dir.pwd, 'storage', path)
  end
end
