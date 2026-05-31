import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../data/audio_ports.dart';
import '../../data/media_cache.dart';

/// Plays a voice message inside a chat bubble: resolves the audio `storage_key`
/// to a local file via the media cache (cache-first, lazy download for history),
/// then offers play/pause + a progress bar + duration on its own player.
class VoiceMessagePlayer extends ConsumerStatefulWidget {
  const VoiceMessagePlayer({
    required this.storageKey,
    this.hintDurationMs = 0,
    super.key,
  });

  final String storageKey;
  final int hintDurationMs;

  @override
  ConsumerState<VoiceMessagePlayer> createState() =>
      _VoiceMessagePlayerState();
}

class _VoiceMessagePlayerState extends ConsumerState<VoiceMessagePlayer> {
  late final AudioPlayerPort _player;
  StreamSubscription<Duration>? _posSub;
  StreamSubscription<bool>? _playSub;

  Duration _position = Duration.zero;
  Duration? _duration;
  bool _playing = false;
  bool _loaded = false;
  bool _loading = false;
  String? _path;

  @override
  void initState() {
    super.initState();
    _player = ref.read(audioPlayerFactoryProvider)();
    _posSub = _player.position().listen((p) {
      if (mounted) {
        setState(() => _position = p);
      }
    });
    _playSub = _player.playing().listen((p) {
      if (mounted) {
        setState(() => _playing = p);
      }
    });
  }

  @override
  void dispose() {
    unawaited(_posSub?.cancel());
    unawaited(_playSub?.cancel());
    unawaited(_player.dispose());
    super.dispose();
  }

  Future<void> _ensureLoaded(String path) async {
    if (_loaded || _loading || _path == path) {
      return;
    }
    _loading = true;
    _path = path;
    try {
      final d = await _player.setFilePath(path);
      if (!mounted) {
        return;
      }
      setState(() {
        _duration = d ?? _player.duration;
        _loaded = true;
        _loading = false;
      });
    } on Object {
      if (mounted) {
        setState(() => _loading = false);
      }
    }
  }

  Future<void> _toggle() async {
    if (!_loaded) {
      return;
    }
    if (_playing) {
      await _player.pause();
      return;
    }
    final dur = _duration;
    if (dur != null && _position >= dur) {
      await _player.seek(Duration.zero);
    }
    await _player.play();
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final async = ref.watch(mediaFileProvider(widget.storageKey));
    return async.when(
      data: (file) {
        WidgetsBinding.instance.addPostFrameCallback(
          (_) => _ensureLoaded(file.path),
        );
        return _bar(theme);
      },
      loading: () => _bar(theme, busy: true),
      error: (_, _) => _bar(theme, broken: true),
    );
  }

  Widget _bar(ThemeData theme, {bool busy = false, bool broken = false}) {
    final total =
        _duration ?? Duration(milliseconds: widget.hintDurationMs);
    final value = (_duration != null && _duration!.inMilliseconds > 0)
        ? (_position.inMilliseconds / _duration!.inMilliseconds).clamp(0.0, 1.0)
        : 0.0;
    final shown = _position > Duration.zero ? _position : total;

    return SizedBox(
      width: 200,
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          IconButton(
            visualDensity: VisualDensity.compact,
            onPressed: (busy || broken || !_loaded) ? null : _toggle,
            icon: Icon(
              broken
                  ? Icons.error_outline
                  : _playing
                  ? Icons.pause_circle_filled
                  : Icons.play_circle_fill,
            ),
            color: theme.colorScheme.primary,
          ),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              mainAxisSize: MainAxisSize.min,
              children: [
                LinearProgressIndicator(
                  value: busy ? null : value,
                  minHeight: 3,
                ),
                const SizedBox(height: 4),
                Text(_fmt(shown), style: theme.textTheme.labelSmall),
              ],
            ),
          ),
        ],
      ),
    );
  }

  static String _fmt(Duration d) {
    final m = d.inMinutes.remainder(60).toString().padLeft(2, '0');
    final s = d.inSeconds.remainder(60).toString().padLeft(2, '0');
    return '$m:$s';
  }
}
