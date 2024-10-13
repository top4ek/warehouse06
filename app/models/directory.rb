# frozen_string_literal: true

class Directory < Sequel::Model
  plugin :validation_helpers
  plugin :tree, key: :parent_id, single_root: false

  one_to_many :documents

  raise_on_save_failure
end
