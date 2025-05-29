# frozen_string_literal: true

module Serializers
  class Node < Blueprinter::Base
    identifier :id
    fields :name, :path, :description, :date

    field :type do |node, _options|
      node.type
    end

    field :size, if: ->(_field_name, node, _options) { node.type == 'file' }
    field :digest, if: ->(_field_name, node, _options) { node.type == 'file' }
  end
end
