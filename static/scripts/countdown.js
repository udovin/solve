/**
 * jQuery Countdown
 *
 * @author Ivan Udovin
 * @license Apache-2.0
 *
 * @version 0.1.0
 */

jQuery.fn.countdown = function (options) {
	// Merge passed options with defaults
	options = jQuery.extend({}, {
		update: function (time) {
			var hours = jQuery(this).children(".hours");
			var minutes = jQuery(this).children(".minutes");
			var seconds = jQuery(this).children(".seconds");

			var updateText = function (element, text) {
				if (element.text() !== text) {
					element.text(text);
				}
			};

			updateText(hours, time.hours);
			updateText(minutes, ("0" + time.minutes).slice(-2));
			updateText(seconds, ("0" + time.seconds).slice(-2));
		},
		finish: function () {},
		delta: 250
	}, options);

	// Helper for parsing time
	var parseTime = function (time) {
		return {
			hours: Math.floor(time / 3600000),
			minutes: Math.floor(time / 60000) % 60,
			seconds: Math.floor(time / 1000) % 60
		};
	};

	var getTime = function () {
		return new Date().getTime();
	};

	return this.each(function () {
		var target = jQuery(this);

		var time = parseInt(target.attr("data-time"), 10) * 1000;
		var start = getTime();

		var update = function () {
			var current = time - Math.round(getTime() - start);
			options.update.call(target.get(0), parseTime(current));

			if (current <= 0) {
				clearInterval(interval);
				options.finish.call(target.get(0));
			}
		};

		var interval = setInterval(update, options.delta);

		update();
	});
};
