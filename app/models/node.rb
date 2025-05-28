# frozen_string_literal: true

class Node < Sequel::Model
  plugin :validation_helpers

  raise_on_save_failure

  # Add this method
  def type
    digest.nil? ? 'directory' : 'file'
  end

  # Consider adding this helper too, if useful for NodeSerializer
  # def full_storage_path
  #   File.join(Dir.pwd, 'storage', path)
  # end
end
