# frozen_string_literal: true

module Serializers
  class Directory < Blueprinter::Base
    identifier :id
    fields :name
  end
end
