# frozen_string_literal: true

module Serializers
  class Node < Blueprinter::Base
    identifier :id # Keep this
    fields :name, :path, :description, :date

    # Add type using the new model method
    field :type do |node, _options|
      node.type
    end

    # Conditional fields for files
    field :size, if: ->(_field_name, node, _options) { node.type == 'file' }
    field :digest, if: ->(_field_name, node, _options) { node.type == 'file' }

    # Optional: if front_matter is stored as a JSON string or similar in the model
    # field :front_matter, if: ->(_field_name, node, _options) { node.type == 'directory' && node.front_matter.present? }
    # For now, description should cover front_matter content from READMEs.
  end
end
