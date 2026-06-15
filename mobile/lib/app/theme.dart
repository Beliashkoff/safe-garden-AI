import 'package:flutter/material.dart';

/// Corner radii of the loft redesign. Generous rounding is the signature of the
/// look: a pill composer, soft message bubbles, rounded cards and sheets.
abstract final class AppRadius {
  static const double composer = 28;
  static const double bubble = 22;
  static const double photo = 18;
  static const double card = 16;
  static const double chip = 14;
  static const double button = 14;
  static const double iconButton = 12;
  static const double sheet = 24;
  static const double dialog = 24;
}

/// The loft palette tokens that Material's [ColorScheme] does not model:
/// the warm "page" background, the soft greys, the user bubble fill, the
/// near-black accent used for the send affordance, and the brand green. Read
/// it via `Theme.of(context).palette` (see [PaletteContext]).
@immutable
class AppPalette extends ThemeExtension<AppPalette> {
  const AppPalette({
    required this.loftBg,
    required this.surface,
    required this.soft,
    required this.softer,
    required this.border,
    required this.borderStrong,
    required this.text,
    required this.textMuted,
    required this.textSubtle,
    required this.bubble,
    required this.accent,
    required this.onAccent,
    required this.brandGreen,
    required this.brandGreenSoft,
    required this.danger,
    required this.inputBg,
    required this.inputBorder,
  });

  /// Warm "page" background behind the white surface (auth, splash, sheets).
  final Color loftBg;

  /// The primary content surface (the chat column itself).
  final Color surface;

  /// Subtle filled backgrounds: suggestion chips, attachment chips, meters.
  final Color soft;

  /// Even subtler fill for inset cards (fertilizer recommendations). The
  /// "что делать" action card uses the stronger [soft] fill so it stands out.
  final Color softer;
  final Color border;
  final Color borderStrong;

  /// Primary text.
  final Color text;

  /// Secondary labels (author row, card captions).
  final Color textMuted;

  /// Tertiary labels (timestamps, separators).
  final Color textSubtle;

  /// The right-aligned user message bubble fill.
  final Color bubble;

  /// Near-black (light) / near-white (dark) — the send button + active states.
  final Color accent;
  final Color onAccent;
  final Color brandGreen;
  final Color brandGreenSoft;
  final Color danger;
  final Color inputBg;
  final Color inputBorder;

  static const light = AppPalette(
    loftBg: Color(0xFFEEECE4),
    surface: Color(0xFFFFFFFF),
    soft: Color(0xFFF7F7F6),
    softer: Color(0xFFFAFAF9),
    border: Color(0x140D0D0D), // rgba(13,13,13,0.08)
    borderStrong: Color(0x240D0D0D), // rgba(13,13,13,0.14)
    text: Color(0xFF0D0D0D),
    textMuted: Color(0xFF6B6B6B),
    textSubtle: Color(0xFF9A9A9A),
    bubble: Color(0xFFF3F3F1),
    accent: Color(0xFF0D0D0D),
    onAccent: Color(0xFFFFFFFF),
    brandGreen: Color(0xFF1F9D4A),
    brandGreenSoft: Color(0x1A1F9D4A), // rgba(31,157,74,0.10)
    danger: Color(0xFFB54A2C),
    inputBg: Color(0xFFFFFFFF),
    inputBorder: Color(0x1F0D0D0D), // rgba(13,13,13,0.12)
  );

  static const dark = AppPalette(
    loftBg: Color(0xFF1A1A1A),
    surface: Color(0xFF0E0E0E),
    soft: Color(0xFF1A1A1A),
    softer: Color(0xFF161616),
    border: Color(0x14FFFFFF), // rgba(255,255,255,0.08)
    borderStrong: Color(0x29FFFFFF), // rgba(255,255,255,0.16)
    text: Color(0xFFECECEC),
    textMuted: Color(0xFF9A9A9A),
    textSubtle: Color(0xFF6B6B6B),
    bubble: Color(0xFF1F1F1F),
    accent: Color(0xFFFFFFFF),
    onAccent: Color(0xFF0D0D0D),
    brandGreen: Color(0xFF3FC56B),
    brandGreenSoft: Color(0x243FC56B), // rgba(63,197,107,0.14)
    danger: Color(0xFFE07A5A),
    inputBg: Color(0xFF1A1A1A),
    inputBorder: Color(0x1AFFFFFF), // rgba(255,255,255,0.10)
  );

  @override
  AppPalette copyWith({
    Color? loftBg,
    Color? surface,
    Color? soft,
    Color? softer,
    Color? border,
    Color? borderStrong,
    Color? text,
    Color? textMuted,
    Color? textSubtle,
    Color? bubble,
    Color? accent,
    Color? onAccent,
    Color? brandGreen,
    Color? brandGreenSoft,
    Color? danger,
    Color? inputBg,
    Color? inputBorder,
  }) {
    return AppPalette(
      loftBg: loftBg ?? this.loftBg,
      surface: surface ?? this.surface,
      soft: soft ?? this.soft,
      softer: softer ?? this.softer,
      border: border ?? this.border,
      borderStrong: borderStrong ?? this.borderStrong,
      text: text ?? this.text,
      textMuted: textMuted ?? this.textMuted,
      textSubtle: textSubtle ?? this.textSubtle,
      bubble: bubble ?? this.bubble,
      accent: accent ?? this.accent,
      onAccent: onAccent ?? this.onAccent,
      brandGreen: brandGreen ?? this.brandGreen,
      brandGreenSoft: brandGreenSoft ?? this.brandGreenSoft,
      danger: danger ?? this.danger,
      inputBg: inputBg ?? this.inputBg,
      inputBorder: inputBorder ?? this.inputBorder,
    );
  }

  @override
  AppPalette lerp(ThemeExtension<AppPalette>? other, double t) {
    if (other is! AppPalette) {
      return this;
    }
    return AppPalette(
      loftBg: Color.lerp(loftBg, other.loftBg, t)!,
      surface: Color.lerp(surface, other.surface, t)!,
      soft: Color.lerp(soft, other.soft, t)!,
      softer: Color.lerp(softer, other.softer, t)!,
      border: Color.lerp(border, other.border, t)!,
      borderStrong: Color.lerp(borderStrong, other.borderStrong, t)!,
      text: Color.lerp(text, other.text, t)!,
      textMuted: Color.lerp(textMuted, other.textMuted, t)!,
      textSubtle: Color.lerp(textSubtle, other.textSubtle, t)!,
      bubble: Color.lerp(bubble, other.bubble, t)!,
      accent: Color.lerp(accent, other.accent, t)!,
      onAccent: Color.lerp(onAccent, other.onAccent, t)!,
      brandGreen: Color.lerp(brandGreen, other.brandGreen, t)!,
      brandGreenSoft: Color.lerp(brandGreenSoft, other.brandGreenSoft, t)!,
      danger: Color.lerp(danger, other.danger, t)!,
      inputBg: Color.lerp(inputBg, other.inputBg, t)!,
      inputBorder: Color.lerp(inputBorder, other.inputBorder, t)!,
    );
  }
}

/// `Theme.of(context).palette` shorthand for the loft tokens.
extension PaletteContext on ThemeData {
  AppPalette get palette => extension<AppPalette>() ?? AppPalette.light;
}

class AppTheme {
  const AppTheme._();

  static const String _font = 'Onest';

  static ThemeData light() => _build(Brightness.light, AppPalette.light);
  static ThemeData dark() => _build(Brightness.dark, AppPalette.dark);

  static ThemeData _build(Brightness brightness, AppPalette p) {
    final scheme =
        ColorScheme.fromSeed(
          seedColor: p.brandGreen,
          brightness: brightness,
        ).copyWith(
          primary: p.brandGreen,
          onPrimary: const Color(0xFFFFFFFF),
          surface: p.surface,
          onSurface: p.text,
          onSurfaceVariant: p.textMuted,
          error: p.danger,
          onError: const Color(0xFFFFFFFF),
          outline: p.borderStrong,
          outlineVariant: p.border,
        );

    final base = ThemeData(
      useMaterial3: true,
      brightness: brightness,
      colorScheme: scheme,
      fontFamily: _font,
      scaffoldBackgroundColor: p.surface,
      splashFactory: InkSparkle.splashFactory,
    );

    return base.copyWith(
      extensions: [p],
      textTheme: _textTheme(base.textTheme, p),
      appBarTheme: AppBarTheme(
        backgroundColor: p.surface,
        surfaceTintColor: Colors.transparent,
        foregroundColor: p.text,
        elevation: 0,
        scrolledUnderElevation: 0.5,
        centerTitle: true,
        titleTextStyle: TextStyle(
          fontFamily: _font,
          fontSize: 16,
          fontWeight: FontWeight.w600,
          letterSpacing: -0.2,
          color: p.text,
        ),
      ),
      cardTheme: CardThemeData(
        color: p.surface,
        surfaceTintColor: Colors.transparent,
        elevation: 0,
        margin: EdgeInsets.zero,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(AppRadius.card),
          side: BorderSide(color: p.border),
        ),
      ),
      dividerTheme: DividerThemeData(
        color: p.border,
        thickness: 0.5,
        space: 0.5,
      ),
      inputDecorationTheme: InputDecorationTheme(
        filled: true,
        fillColor: p.inputBg,
        hintStyle: TextStyle(color: p.textSubtle, fontFamily: _font),
        contentPadding: const EdgeInsets.symmetric(
          horizontal: 16,
          vertical: 14,
        ),
        border: OutlineInputBorder(
          borderRadius: BorderRadius.circular(AppRadius.card),
          borderSide: BorderSide(color: p.inputBorder),
        ),
        enabledBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(AppRadius.card),
          borderSide: BorderSide(color: p.inputBorder),
        ),
        focusedBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(AppRadius.card),
          borderSide: BorderSide(color: p.accent, width: 1.5),
        ),
        errorBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(AppRadius.card),
          borderSide: BorderSide(color: p.danger),
        ),
        focusedErrorBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(AppRadius.card),
          borderSide: BorderSide(color: p.danger, width: 1.5),
        ),
      ),
      filledButtonTheme: FilledButtonThemeData(
        style: FilledButton.styleFrom(
          minimumSize: const Size.fromHeight(52),
          textStyle: const TextStyle(
            fontFamily: _font,
            fontSize: 16,
            fontWeight: FontWeight.w600,
          ),
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(AppRadius.button),
          ),
        ),
      ),
      outlinedButtonTheme: OutlinedButtonThemeData(
        style: OutlinedButton.styleFrom(
          minimumSize: const Size.fromHeight(52),
          foregroundColor: p.text,
          side: BorderSide(color: p.borderStrong),
          textStyle: const TextStyle(
            fontFamily: _font,
            fontSize: 16,
            fontWeight: FontWeight.w600,
          ),
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(AppRadius.button),
          ),
        ),
      ),
      textButtonTheme: TextButtonThemeData(
        style: TextButton.styleFrom(
          foregroundColor: p.accent,
          textStyle: const TextStyle(
            fontFamily: _font,
            fontWeight: FontWeight.w600,
          ),
        ),
      ),
      dialogTheme: DialogThemeData(
        backgroundColor: p.surface,
        surfaceTintColor: Colors.transparent,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(AppRadius.dialog),
        ),
        titleTextStyle: TextStyle(
          fontFamily: _font,
          fontSize: 18,
          fontWeight: FontWeight.w600,
          color: p.text,
        ),
        contentTextStyle: TextStyle(
          fontFamily: _font,
          fontSize: 15,
          height: 1.4,
          color: p.textMuted,
        ),
      ),
      bottomSheetTheme: BottomSheetThemeData(
        backgroundColor: p.surface,
        surfaceTintColor: Colors.transparent,
        showDragHandle: true,
        shape: const RoundedRectangleBorder(
          borderRadius: BorderRadius.vertical(
            top: Radius.circular(AppRadius.sheet),
          ),
        ),
      ),
      snackBarTheme: SnackBarThemeData(
        behavior: SnackBarBehavior.floating,
        backgroundColor: p.accent,
        contentTextStyle: TextStyle(color: p.onAccent, fontFamily: _font),
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(AppRadius.chip),
        ),
      ),
      listTileTheme: ListTileThemeData(
        iconColor: p.textMuted,
        textColor: p.text,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(AppRadius.chip),
        ),
      ),
      drawerTheme: DrawerThemeData(
        backgroundColor: p.surface,
        surfaceTintColor: Colors.transparent,
        shape: const RoundedRectangleBorder(
          borderRadius: BorderRadius.horizontal(right: Radius.circular(28)),
        ),
      ),
      progressIndicatorTheme: ProgressIndicatorThemeData(color: p.brandGreen),
      iconTheme: IconThemeData(color: p.text),
    );
  }

  /// Onest text scale tuned to the reference: tight letter-spacing on the
  /// larger sizes, calm body line-height. Colours come from [p].
  static TextTheme _textTheme(TextTheme base, AppPalette p) {
    final t = base.apply(
      bodyColor: p.text,
      displayColor: p.text,
      fontFamily: _font,
    );
    return t.copyWith(
      headlineMedium: t.headlineMedium?.copyWith(
        fontSize: 24,
        fontWeight: FontWeight.w600,
        letterSpacing: -0.5,
        height: 1.2,
      ),
      titleLarge: t.titleLarge?.copyWith(
        fontSize: 18,
        fontWeight: FontWeight.w600,
        letterSpacing: -0.3,
        height: 1.3,
      ),
      titleMedium: t.titleMedium?.copyWith(
        fontSize: 16,
        fontWeight: FontWeight.w600,
        letterSpacing: -0.2,
      ),
      bodyLarge: t.bodyLarge?.copyWith(
        fontSize: 15.5,
        height: 1.5,
        letterSpacing: -0.1,
      ),
      bodyMedium: t.bodyMedium?.copyWith(fontSize: 14.5, height: 1.45),
      bodySmall: t.bodySmall?.copyWith(fontSize: 13, color: p.textMuted),
      labelLarge: t.labelLarge?.copyWith(
        fontSize: 15,
        fontWeight: FontWeight.w600,
      ),
      labelSmall: t.labelSmall?.copyWith(fontSize: 12, color: p.textSubtle),
    );
  }
}
