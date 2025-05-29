# lib/renderers/custom_html_renderer.rb
# frozen_string_literal: true

require 'redcarpet'

class CustomHtmlRenderer < Redcarpet::Render::HTML
  def initialize(node_path, options = {}) # node_path is the path of the markdown file
    super(options)
    @node_path = node_path
  end

  def image(link, title, alt_text)
    if !(link.start_with?('http://') || link.start_with?('https://') || link.start_with?('/'))
      base_dir = File.dirname(@node_path)
      new_link = if base_dir == '.' || base_dir == '/'
                   "/#{link}"
                 else
                   "/#{File.join(base_dir, link)}"
                 end
      super(new_link, title, alt_text)
    else
      super(link, title, alt_text)
    end
  end
end
