# frozen_string_literal: true

module Interactors
  module Document
    class Regenerate
      class << self
        def call(_request)
          RebuildDbService.new.call.size
        end
      end
    end
  end
end
