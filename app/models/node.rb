# frozen_string_literal: true

class Node < Sequel::Model
  plugin :validation_helpers

  raise_on_save_failure
end
