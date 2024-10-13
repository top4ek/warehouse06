# frozen_string_literal: true

module Routes
  class Api < Base
    route do |r|
      r.get 'regenerate' do
        { size: Interactors::Document::Regenerate.call(r) }
      end
    end
  end
end
