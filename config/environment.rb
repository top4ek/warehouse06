# config/environment.rb
ENV['RACK_ENV'] ||= 'development' # Default for general use

require 'bundler/setup'
Bundler.require(:default, ENV['RACK_ENV'].to_sym)

# Ensure .env files are loaded. This is crucial before any app code relies on ENV vars.
# If RACK_ENV is 'test' (set by spec_helper), this will load .env.test.
require 'dotenv/load'

project_root_config = File.expand_path('..', __dir__) # This is <project_root>/config

# --- Files that need to be loaded before Zeitwerk ---
# Extensions, initializers, or configurations that don't follow Zeitwerk's conventions
# or need to be available very early.
require File.join(project_root_config, 'config/extensions/array/wrap.rb')
require File.join(project_root_config, 'config/database.rb') # Establishes DB connection

# --- Zeitwerk Setup ---
require 'zeitwerk'
loader = Zeitwerk::Loader.new

# Define the root directory for Zeitwerk.
# Since project_root_config is <project_root>/config, we need to go one level up.
app_root = File.expand_path('..', project_root_config)

# Add directories to be managed by Zeitwerk.
# Zeitwerk expects standard naming conventions (e.g., app/models/user.rb defines User or Models::User).
loader.push_dir(File.join(app_root, 'app'))
# If you had a 'lib' directory you wanted Zeitwerk to manage:
# loader.push_dir(File.join(app_root, 'lib'))

# Optional: Collapse specific directories if their structure doesn't map to namespaces.
# For example, if files in 'app/models' should define top-level constants (e.g., Node)
# rather than namespaced (e.g., Models::Node), you might need:
# loader.collapse(File.join(app_root, 'app/models'))
# loader.collapse(File.join(app_root, 'app/services'))
# ... and so on for other subdirectories of 'app'.
# This is common if your app isn't heavily modularized with namespaces.
# Given the current structure (e.g., app/models.rb requiring ./models/*),
# the files likely define top-level constants (Node, Tag, etc.).
# So, collapsing these directories is probably necessary.

# Determine which directories under 'app' exist and should be collapsed.
# Common ones from this project:
app_subdirs_to_collapse = %w[models contracts services interactors serializers routes]
app_subdirs_to_collapse.each do |subdir|
  dir_path = File.join(app_root, 'app', subdir)
  loader.collapse(dir_path) if Dir.exist?(dir_path)
end


# Enable reloading with Zeitwerk if you want to use its reloading capabilities
# This is often used in conjunction with a web server that supports it, or custom reloading logic.
# For now, we are planning to use `rerun` which restarts the whole process,
# so Zeitwerk's own reloader might not be strictly necessary but can be enabled.
# It's generally safe to enable if RACK_ENV is development.
if ENV['RACK_ENV'] == 'development'
  loader.enable_reloading
end

loader.setup # Finalize Zeitwerk setup. Constants are now autoloadable.

# --- Files/code to run AFTER Zeitwerk setup (if any) ---
# For example, if some code needs all constants to be loaded.

# The previous explicit requires for:
# require File.join(project_root_config, 'app/models.rb')
# require File.join(project_root_config, 'app/contracts.rb')
# ...etc. for files under 'app/'...
# are now handled by Zeitwerk and should be REMOVED.
