# frozen_string_literal: true

module Serializers
  class Document < Blueprinter::Base
    identifier :id
    fields :name
  end
end
