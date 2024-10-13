# frozen_string_literal: true

class Document < Sequel::Model
  plugin :validation_helpers

  many_to_one :directory

  raise_on_save_failure
end
