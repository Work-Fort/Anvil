# Configuration file for the Sphinx documentation builder.
#
# For the full list of built-in configuration values, see the documentation:
# https://www.sphinx-doc.org/en/master/usage/configuration.html

# -- Project information -----------------------------------------------------
# https://www.sphinx-doc.org/en/master/usage/configuration.html#project-information

project = 'Cracker Barrel'
copyright = '2026, Kaz Walker'
author = 'Kaz Walker'

version = '1.0'
release = '1.0'

# -- General configuration ---------------------------------------------------
# https://www.sphinx-doc.org/en/master/usage/configuration.html#general-configuration

extensions = [
    'sphinx_rtd_theme',
    'sphinx_tabs.tabs',
    'sphinx_copybutton',
    'sphinx_togglebutton',
    'sphinxcontrib.jquery',
]

templates_path = ['_templates']
exclude_patterns = []

language = 'en'

# -- Options for HTML output -------------------------------------------------
# https://www.sphinx-doc.org/en/master/usage/configuration.html#options-for-html-output

html_theme = 'sphinx_rtd_theme'
html_static_path = ['_static']
html_extra_path = ['install-anvil.sh']

html_theme_options = {
    'logo_only': False,
    'prev_next_buttons_location': 'bottom',
    'navigation_depth': 4,
    'collapse_navigation': False,
    'sticky_navigation': True,
}

# -- Extension configuration -------------------------------------------------

# sphinx-copybutton: Don't copy prompts
copybutton_prompt_text = r">>> |\.\.\. |\$ "
copybutton_prompt_is_regexp = True

# -- Linkcheck configuration -------------------------------------------------

# Ignore URLs that block automated requests (403 Forbidden)
linkcheck_ignore = [
    r'https://www\.npmjs\.com/.*',
]


def setup(app):
    # Theme customizations (Zephyr-style)
    # Note: dark.css and light.css are loaded via templates/layout.html with media queries
    # This allows the dark-mode-toggle component to work properly
    app.add_css_file("css/custom.css")
    app.add_js_file("js/custom.js")
