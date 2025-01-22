# frozen_string_literal: true

module Serializers
  class Node < Blueprinter::Base
    identifier :id
    fields :name
  end
end
