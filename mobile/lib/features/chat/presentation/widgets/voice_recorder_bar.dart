import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../l10n/generated/app_localizations.dart';
import '../../application/voice_recorder_controller.dart';
import '../../data/audio_ports.dart';

/// Replaces the text input while a voice message is being recorded or previewed.
/// Recording shows a timer + level meter + cancel hint; preview shows a local
/// player with Delete / Send. The hold gesture itself lives on the mic button in
/// the chat screen — this bar is display + the preview actions.
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
    final canceling = state.cancelArmed;
    final accent = canceling ? theme.colorScheme.error : theme.colorScheme.primary;
    return Row(
      children: [
        Icon(Icons.fiber_manual_record, color: theme.colorScheme.error, size: 14),
        const SizedBox(width: 8),
        Text(_fmt(state.elapsedMs), style: theme.textTheme.titleMedium),
        const SizedBox(width: 12),
        Expanded(child: _LevelMeter(level: state.amplitude, color: accent)),
        const SizedBox(width: 12),
        Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(Icons.chevron_left, size: 16, color: accent),
            Text(
              canceling ? l10n.voiceReleaseToCancel : l10n.voiceSlideToCancel,
              style: theme.textTheme.bodySmall?.copyWith(color: accent),
            ),
          ],
        ),
      ],
    );
  }

  Widget _previewRow(
    ThemeData theme,
    AppLocalizations l10n,
    VoiceRecorderState state,
  ) {
    final uploading = state.phase == VoicePhase.uploading;
    return Row(
      children: [
        IconButton(
          tooltip: l10n.commonDelete,
          onPressed: uploading
              ? null
              : () => ref.read(voiceRecorderProvider.notifier).discardPreview(),
          icon: const Icon(Icons.delete_outline),
        ),
        IconButton(
          onPressed: uploading ? null : _togglePreview,
          icon: Icon(
            _playing ? Icons.pause_circle_filled : Icons.play_circle_fill,
          ),
          color: theme.colorScheme.primary,
        ),
        Expanded(
          child: Text(_fmt(state.elapsedMs), style: theme.textTheme.labelLarge),
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
          IconButton.filled(
            tooltip: l10n.chatSend,
            onPressed: _send,
            icon: const Icon(Icons.send_rounded),
          ),
      ],
    );
  }

  static String _fmt(int ms) {
    final d = Duration(milliseconds: ms);
    final m = d.inMinutes.remainder(60).toString().padLeft(2, '0');
    final s = d.inSeconds.remainder(60).toString().padLeft(2, '0');
    return '$m:$s';
  }
}

/// A simple horizontal level meter that fills with the normalised [level].
class _LevelMeter extends StatelessWidget {
  const _LevelMeter({required this.level, required this.color});

  final double level;
  final Color color;

  @override
  Widget build(BuildContext context) {
    return ClipRRect(
      borderRadius: BorderRadius.circular(3),
      child: LinearProgressIndicator(
        value: level.clamp(0.0, 1.0),
        minHeight: 6,
        backgroundColor: Theme.of(context).colorScheme.surfaceContainerHighest,
        valueColor: AlwaysStoppedAnimation<Color>(color),
      ),
    );
  }
}
