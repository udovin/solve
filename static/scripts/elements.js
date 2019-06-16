/**
 * @author Ivan Udovin
 * @license MIT
 */

"use strict";

/*
 * Fix bug with animations
 */
jQuery(document).ready(function () {
	jQuery(document.body).removeClass("preload");
});

/*
 * Enable pretty selects
 */
jQuery(document).ready(function () {
	jQuery("select.ui-select").each(function () {
		var $this = jQuery(this);

		var select = jQuery(document.createElement("span"))
			.addClass("ui-select");

		var button = jQuery(document.createElement("button"))
			.addClass("ui-select-button")
			.attr("type", "button");

		var arrow = jQuery(document.createElement("span"))
			.addClass("ui-select-arrow");

		var options = jQuery(document.createElement("div"))
			.addClass("ui-select-options");

		var optionList = jQuery(document.createElement("ul"))
			.appendTo(options);

		var selectOption = function () {
			var $option = jQuery(this);

			optionList
				.children("li.selected")
				.removeClass("selected");

			$option.addClass("selected");

			button.text($option.text());
			$this.val($option.data("value")).change();
		};

		$this.children("option").each(function () {
			var $this = jQuery(this);

			var option = jQuery(document.createElement("li"))
				.text($this.text())
				.data("value", $this.attr("value"));

			if ($this.is(":disabled")) {
				option.addClass("disabled");
			} else if ($this.is(":selected")) {
				selectOption.call(option);
			}

			option.click(selectOption);
			optionList.append(option);
		});

		var layer = $this.parents(".ui-layer");

		button.click(function () {
			if (!select.hasClass("focus")) {
				var offset = button.offset();

				options.stop().hide().appendTo(layer).css({
					top: offset.top + layer.scrollTop(),
					left: offset.left + layer.scrollLeft(),
					minWidth: button.outerWidth()
				}).fadeIn(150);
			}
		});

		jQuery(document).click(function (event) {
			var $target = jQuery(event.target);

			if ($target.is(button) && !select.hasClass("focus")) {
				select.addClass("focus");

				return;
			}

			select.removeClass("focus");

			options.stop().fadeOut(150, function () {
				options.detach();
			});
		});

		$this.before(select.append(button, arrow)).hide();
	});
});

/*
 * Enable pretty checkboxes
 */
jQuery(document).ready(function () {
	jQuery("input.ui-checkbox").each(function () {
		var $this = jQuery(this);

		var checkbox = jQuery(document.createElement("button"))
			.addClass("ui-checkbox")
			.attr("type", "button");

		var check = jQuery(document.createElement("span"))
			.addClass("ui-checkbox-check");

		if ($this.is(":checked")) {
			checkbox.addClass("checked");
		}

		checkbox.click(function () {
			checkbox.toggleClass("checked");
			$this.attr("checked", checkbox.hasClass("checked"));
		});

		$this.before(checkbox.append(check)).hide();
	});
});

/*
 * Enable math expressions
 */
jQuery(document).ready(function () {
	jQuery(".ui-math").each(function () {
		var $this = jQuery(this);

		katex.render($this.text(), this);
	});
});

/*
 * Enable tooltip support
 */
jQuery(document).ready(function () {
	jQuery("[title]").each(function () {
		var $this = jQuery(this);

		// Create all needed elements
		var wrap = jQuery(document.createElement("div"))
			.addClass("ui-tooltip-wrap");
		var tooltip = jQuery(document.createElement("div"))
			.addClass("ui-tooltip");
		var content = jQuery(document.createElement("div"))
			.addClass("ui-tooltip-content");
		var arrow = jQuery(document.createElement("div"))
			.addClass("ui-tooltip-arrow");

		// Initialize structure
		wrap.append(arrow, tooltip.append(content));

		var showTooltip = function () {
			content.text(jQuery(this).attr("title"));

			$this.attr("title", null);

			var layer = $this.parents(".ui-layer");

			wrap.stop().hide().appendTo(layer).css({
				top: $this.offset().top + layer.scrollTop() +
					$this.outerHeight(),
				left: $this.offset().left + layer.scrollLeft() +
					($this.outerWidth() - wrap.outerWidth()) / 2
			}).fadeIn(150);
		};

		var hideTooltip = function () {
			wrap.stop().fadeOut(150, function () {
				wrap.detach();
			});

			$this.attr("title", content.text());
		};

		$this.mouseenter(showTooltip).mouseleave(hideTooltip);
	});
});

/*
 * Enable code highlight
 */
jQuery(document).ready(function () {
	CodeMirror.modeURL =
		"/assets/library/codemirror/mode/%N.js";

	var setHighlightMode = function (highlight, mode) {
		var modeInfo = CodeMirror.findModeByExtension(mode);

		highlight.setOption("mode", "text/plain");

		if (modeInfo) {
			highlight.setOption("mode", modeInfo.mime);
			CodeMirror.autoLoadMode(highlight, modeInfo.mode);
		}
	};

	jQuery(".ui-code").each(function () {
		var $this = jQuery(this);
		var $code = $this.find("code");

		$this.children().hide();

		var highlight = CodeMirror(
			this,
			{
				value: $code.text(),
				readOnly: true,
				lineNumbers: $this.hasClass('with-numbers'),
				cursorBlinkRate: -1,
				mode: "text/plain"
			}
		);

		highlight.setMode = function (mode) {
			setHighlightMode(highlight, mode);
		};

		if ($this.attr("data-type")) {
			highlight.setMode($this.attr("data-type"));
		}

		$this.data("highlight", highlight);
	});

	jQuery(".ui-editor").each(function () {
		var $this = jQuery(this);
		var $textarea = $this.find("textarea");

		var highlight = CodeMirror.fromTextArea(
			$textarea.get(0),
			{
				lineNumbers: true,
				indentWithTabs: true,
				indentUnit: 4,
				mode: "text/plain"
			}
		);

		highlight.setMode = function (mode) {
			setHighlightMode(highlight, mode);
		};

		if ($this.attr("data-type")) {
			highlight.setMode($this.attr("data-type"));
		}

		$this.data("highlight", highlight);
	});

	jQuery(".ui-countdown").each(function () {
		var $this = jQuery(this);

		$this.countdown();
	});
});
