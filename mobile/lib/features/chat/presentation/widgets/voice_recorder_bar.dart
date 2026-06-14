import 'dart:async';
import 'dart:math' as math;

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../app/theme.dart';
import '../../../../l10n/generated/app_localizations.dart';
import '../../application/voice_recorder_controller.dart';
import '../../data/audio_ports.dart';

/// Replaces the text input while a voice message is being recorded or previewed.
/// Recording shows a timer + animated waveform + cancel hint; preview shows a
/// local player with Delete / Send. The hold gesture itself lives on the mic
/// button in the chat screen — this bar is display + the preview actions.
class VoiceRecorderBar extends ConsumerStatefulWidget {
  const VoiceRecorderBar({super.key});

  @override
  ConsumerState<VoiceRecorderBar> createState() => _VoiceRecorderBarState();
}

class _VoiceRecorderBarState extends ConsumerState<VoiceRecorderBar> {
  AudioPlayerPort? _player;
  StreamSubscription<bool>? _playSub;
  bool _playing = false;
  String? _loadedPath;

  @override
  void dispose() {
    unawaited(_playSub?.cancel());
    unawaited(_player?.dispose());
    super.dispose();
  }

  Future<void> _loadPreview(String path) async {
    if (_loadedPath == path) {
      return;
    }
    _loadedPath = path;
    final player = _player ??= ref.read(audioPlayerFactoryProvider)();
    _playSub ??= player.playing().listen((v) {
      if (mounted) {
        setState(() => _playing = v);
      }
    });
    try {
      await player.setFilePath(path);
    } on Object {
      // ignore — the player just won't be playable
    }
  }

  Future<void> _togglePreview() async {
    final player = _player;
    if (player == null) {
      return;
    }
    if (_playing) {
      await player.pause();
    } else {
      await player.play();
    }
  }

  Future<void> _send() async {
    final messenger = ScaffoldMessenger.of(context);
    final l10n = AppLocalizations.of(context)!;
    final ok = await ref.read(voiceRecorderProvider.notifier).sendPreview();
    if (!ok && mounted) {
      messenger.showSnackBar(SnackBar(content: Text(l10n.voiceUploadFailed)));
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    final state = ref.watch(voiceRecorderProvider);

    switch (state.phase) {
      case VoicePhase.recording:
        return _recordingRow(theme, l10n, state);
      case VoicePhase.preview:
      case VoicePhase.uploading:
        WidgetsBinding.instance.addPostFrameCallback((_) {
          final path = state.filePath;
          if (path != null) {
            _loadPreview(path);
          }
        });
        return _previewRow(theme, l10n, state);
      case VoicePhase.idle:
        return const SizedBox.shrink();
    }
  }

  Widget _recordingRow(
    ThemeData theme,
    AppLocalizations l10n,
    VoiceRecorderState state,
  ) {
    final p = theme.palette;
    final canceling = state.cancelArmed;
    final hintColor = canceling ? p.danger : p.textMuted;
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 8),
      child: Row(
        children: [
          _RecordDot(color: p.danger),
          const SizedBox(width: 8),
          Text(
            _fmt(state.elapsedMs),
            style: theme.textTheme.bodyMedium?.copyWith(
              fontFeatures: const [FontFeature.tabularFigures()],
              fontWeight: FontWeight.w500,
            ),
          ),
          const SizedBox(width: 12),
          Expanded(
            child: _Waveform(
              level: state.amplitude,
              color: canceling ? p.danger : p.text,
              idle: p.textSubtle,
            ),
          ),
          const SizedBox(width: 10),
          Icon(Icons.chevron_left_rounded, size: 16, color: hintColor),
          Text(
            canceling ? l10n.voiceReleaseToCancel : l10n.voiceSlideToCancel,
            style: theme.textTheme.bodySmall?.copyWith(color: hintColor),
          ),
        ],
      ),
    );
  }

  Widget _previewRow(
    ThemeData theme,
    AppLocalizations l10n,
    VoiceRecorderState state,
  ) {
    final p = theme.palette;
    final uploading = state.phase == VoicePhase.uploading;
    return Row(
      children: [
        _softCircle(
          icon: Icons.delete_outline_rounded,
          color: p.textMuted,
          bg: p.soft,
          onTap: uploading
              ? null
              : () => ref.read(voiceRecorderProvider.notifier).discardPreview(),
          tooltip: l10n.commonDelete,
        ),
        const SizedBox(width: 6),
        _softCircle(
          icon: _playing
              ? Icons.pause_rounded
              : Icons.play_arrow_rounded,
          color: p.text,
          bg: p.soft,
          onTap: uploading ? null : _togglePreview,
        ),
        const SizedBox(width: 12),
        Expanded(
          child: Text(
            _fmt(state.elapsedMs),
            style: theme.textTheme.bodyMedium?.copyWith(
              fontFeatures: const [FontFeature.tabularFigures()],
            ),
          ),
        ),
        if (uploading)
          const Padding(
            padding: EdgeInsets.all(8),
            child: SizedBox(
              height: 20,
              width: 20,
              child: CircularProgressIndicator(strokeWidth: 2),
            ),
          )
        else
          _softCircle(
            icon: Icons.arrow_upward_rounded,
            color: p.onAccent,
            bg: p.accent,
            onTap: _send,
            tooltip: l10n.chatSend,
          ),
      ],
    );
  }

  Widget _softCircle({
    required IconData icon,
    required Color color,
    required Color bg,
    required VoidCallback? onTap,
    String? tooltip,
  }) {
    final btn = Material(
      color: bg,
      shape: const CircleBorder(),
      child: InkWell(
        onTap: onTap,
        customBorder: const CircleBorder(),
        child: SizedBox(
          width: 38,
          height: 38,
          child: Icon(icon, color: color, size: 20),
        ),
      ),
    );
    return tooltip == null ? btn : Tooltip(message: tooltip, child: btn);
  }

  static String _fmt(int ms) {
    final d = Duration(milliseconds: ms);
    final m = d.inMinutes.remainder(60).toString().padLeft(2, '0');
    final s = d.inSeconds.remainder(60).toString().padLeft(2, '0');
    return '$m:$s';
  }
}

/// A pulsing red record indicator.
class _RecordDot extends StatefulWidget {
  const _RecordDot({required this.color});

  final Color color;

  @override
  State<_RecordDot> createState() => _RecordDotState();
}

class _RecordDotState extends State<_RecordDot>
    with SingleTickerProviderStateMixin {
  late final AnimationController _c = AnimationController(
    vsync: this,
    duration: const Duration(milliseconds: 1400),
  )..repeat(reverse: true);

  @override
  void dispose() {
    _c.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return FadeTransition(
      opacity: Tween<double>(begin: 1, end: 0.35).animate(_c),
      child: Container(
        width: 9,
        height: 9,
        decoration: BoxDecoration(color: widget.color, shape: BoxShape.circle),
      ),
    );
  }
}

/// Animated waveform driven by the live amplitude and a free-running clock,
/// echoing the reference recorder. Bars to the left of the playhead use the
/// active colour, the rest the idle colour.
class _Waveform extends StatefulWidget {
  const _Waveform({required this.level, required this.color, required this.idle});

  final double level;
  final Color color;
  final Color idle;

  @override
  State<_Waveform> createState() => _WaveformState();
}

class _WaveformState extends State<_Waveform>
    with SingleTickerProviderStateMixin {
  late final AnimationController _c = AnimationController(
    vsync: this,
    duration: const Duration(seconds: 4),
  )..repeat();

  static const _bars = 28;

  @override
  void dispose() {
    _c.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return SizedBox(
      height: 24,
      child: AnimatedBuilder(
        animation: _c,
        builder: (context, _) {
          final t = _c.value * _bars * 2;
          final level = widget.level.clamp(0.0, 1.0);
          return Row(
            mainAxisAlignment: MainAxisAlignment.spaceBetween,
            crossAxisAlignment: CrossAxisAlignment.center,
            children: List.generate(_bars, (i) {
              final wave = math.sin((i + t) * 0.6 + i).abs();
              final h = 3 + (wave * 14 * (0.35 + level)).clamp(0.0, 17.0);
              final active = i < (t.toInt() % _bars);
              return Container(
                width: 2.5,
                height: h,
                decoration: BoxDecoration(
                  color: active ? widget.color : widget.idle,
                  borderRadius: BorderRadius.circular(2),
                ),
              );
            }),
          );
        },
      ),
    );
  }
}
