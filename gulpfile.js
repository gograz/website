var Gulp = require('gulp'),
    Sass = require('gulp-sass'),
    Rev = require('gulp-rev');

var themePrefix = 'themes/gograz/';

Gulp.task('default', ['sass']);

Gulp.task('sass', function() {
    Gulp.src(themePrefix + 'static/sass/*.scss')
        .pipe(Sass())
        .pipe(Rev())
        .pipe(Gulp.dest(themePrefix + 'static/css'))
        .pipe(Rev.manifest())
        .pipe(Gulp.dest(themePrefix + 'static/css'));
});

Gulp.task('watch', function() {
    Gulp.watch(['themePrefix + static/sass/*.scss'], ['sass']);
});
