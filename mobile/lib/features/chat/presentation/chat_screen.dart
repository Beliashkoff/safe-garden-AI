import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:url_launcher/url_launcher.dart';

import '../../../app/theme.dart';
import '../../../app/widgets/brand_logo.dart';
import '../../../core/config/app_config.dart';
import '../../../l10n/generated/app_localizations.dart';
import '../../auth/application/auth_controller.dart';
import '../../auth/presentation/auth_error_message.dart';
import '../application/chat_controller.dart';
import '../application/message_composer.dart';
import '../application/voice_recorder_controller.dart';
import '../data/media_ports.dart';
import '../domain/chat_models.dart';
import 'chat_error_message.dart';
import 'message_bubble.dart';
import 'widgets/attachment_strip.dart';
import 'widgets/chat_drawer.dart';
import 'widgets/empty_state.dart';
import 'widgets/voice_recorder_bar.dart';

enum _AttachSource { camera, gallery }

class ChatScreen extends ConsumerStatefulWidget {
  const ChatScreen({super.key});

  @override
  ConsumerState<ChatScreen> createState() => _ChatScreenState();
}

class _ChatScreenState extends ConsumerState<ChatScreen> {
  final _input = TextEditingController();
  final _scroll = ScrollController();
  final _inputFocus = FocusNode();
  final _scaffoldKey = GlobalKey<ScaffoldState>();

  @override
  void initState() {
    super.initState();
    _scroll.addListener(_onScroll);
  }

  @override
  void dispose() {
    _scroll.removeListener(_onScroll);
    _input.dispose();
    _scroll.dispose();
    _inputFocus.dispose();
    super.dispose();
  }

  void _onScroll() {
    // reverse:true → older messages sit at the top, i.e. max scroll extent.
    if (_scroll.position.pixels >= _scroll.position.maxScrollExtent - 200) {
      ref.read(chatControllerProvider.notifier).loadOlder();
    }
  }

  void _scrollToLatest() {
    if (_scroll.hasClients) {
      _scroll.animateTo(
        0,
        duration: const Duration(milliseconds: 250),
        curve: Curves.easeOut,
      );
    }
  }

  Future<void> _send() async {
    final text = _input.text.trim();
    final composer = ref.read(messageComposerProvider);
    if (composer.uploading) {
      return;
    }
    if (text.isEmpty && !composer.hasAttachments) {
      return;
    }
    final l10n = AppLocalizations.of(context)!;
    final messenger = ScaffoldMessenger.of(context);
    _input.clear();
    if (composer.hasAttachments) {
      final ok = await ref
          .read(messageComposerProvider.notifier)
          .uploadAndSend(text);
      if (!ok && mounted) {
        messenger.showSnackBar(SnackBar(content: Text(l10n.chatUploadFailed)));
      }
      return;
    }
    await ref.read(chatControllerProvider.notifier).sendMessage(text);
  }

  void _onSuggestion(ChatSuggestion s) {
    switch (s) {
      case ChatSuggestion.disease:
        _showAttachSheet();
      case ChatSuggestion.deficiency:
        _input.text = AppLocalizations.of(context)!.chatSuggestDeficiency;
        _inputFocus.requestFocus();
      case ChatSuggestion.plan:
        _input.text = AppLocalizations.of(context)!.chatSuggestPlan;
        _inputFocus.requestFocus();
    }
  }

  Future<void> _showAttachSheet() async {
    final l10n = AppLocalizations.of(context)!;
    final source = await showModalBottomSheet<_AttachSource>(
      context: context,
      builder: (ctx) => SafeArea(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            ListTile(
              leading: const Icon(Icons.photo_camera_outlined),
              title: Text(l10n.chatAttachCamera),
              onTap: () => Navigator.of(ctx).pop(_AttachSource.camera),
            ),
            ListTile(
              leading: const Icon(Icons.photo_library_outlined),
              title: Text(l10n.chatAttachGallery),
              onTap: () => Navigator.of(ctx).pop(_AttachSource.gallery),
            ),
            const SizedBox(height: 8),
          ],
        ),
      ),
    );
    if (source == null || !mounted) {
      return;
    }
    final notifier = ref.read(messageComposerProvider.notifier);
    final result = source == _AttachSource.camera
        ? await notifier.addFromCamera()
        : await notifier.addFromGallery();
    if (!mounted) {
      return;
    }
    _handleAttachResult(result, source);
  }

  void _handleAttachResult(AttachRequestResult result, _AttachSource source) {
    final l10n = AppLocalizations.of(context)!;
    final messenger = ScaffoldMessenger.of(context);
    switch (result) {
      case AttachRequestResult.added:
      case AttachRequestResult.cancelled:
        break;
      case AttachRequestResult.limitReached:
        messenger.showSnackBar(
          SnackBar(content: Text(l10n.chatMaxPhotos(kMaxPhotosPerMessage))),
        );
      case AttachRequestResult.permissionDenied:
      case AttachRequestResult.permissionPermanentlyDenied:
        _showPermissionDialog(source);
      case AttachRequestResult.failed:
        messenger.showSnackBar(SnackBar(content: Text(l10n.chatErrorGeneric)));
    }
  }

  Future<void> _showPermissionDialog(_AttachSource source) async {
    final l10n = AppLocalizations.of(context)!;
    final isCamera = source == _AttachSource.camera;
    final open = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text(
          isCamera
              ? l10n.chatPermissionCameraTitle
              : l10n.chatPermissionPhotosTitle,
        ),
        content: Text(
          isCamera
              ? l10n.chatPermissionCameraBody
              : l10n.chatPermissionPhotosBody,
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(ctx).pop(false),
            child: Text(l10n.commonCancel),
          ),
          FilledButton(
            onPressed: () => Navigator.of(ctx).pop(true),
            child: Text(l10n.chatOpenSettings),
          ),
        ],
      ),
    );
    if (open == true) {
      await ref.read(permissionPortProvider).openSettings();
    }
  }

  Future<void> _logout() => ref.read(authControllerProvider.notifier).logout();

  Future<void> _openUrl(String url) async {
    final uri = Uri.tryParse(url);
    if (uri != null) {
      await launchUrl(uri, mode: LaunchMode.externalApplication);
    }
  }

  Future<void> _confirmDeleteAccount() async {
    final l10n = AppLocalizations.of(context)!;
    final messenger = ScaffoldMessenger.of(context);
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text(l10n.deleteAccountConfirmTitle),
        content: Text(l10n.deleteAccountConfirmBody),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(ctx).pop(false),
            child: Text(l10n.commonCancel),
          ),
          FilledButton(
            style: FilledButton.styleFrom(
              backgroundColor: Theme.of(ctx).colorScheme.error,
            ),
            onPressed: () => Navigator.of(ctx).pop(true),
            child: Text(l10n.commonDelete),
          ),
        ],
      ),
    );
    if (confirmed != true) {
      return;
    }
    try {
      await ref.read(authControllerProvider.notifier).deleteAccount();
    } on Object catch (e) {
      messenger.showSnackBar(
        SnackBar(content: Text(authErrorMessage(l10n, e))),
      );
    }
  }

  Future<void> _confirmDeleteMessage(ChatMessage message) async {
    if (message.id.startsWith('local-')) {
      return; // optimistic message, not yet persisted on the server
    }
    final l10n = AppLocalizations.of(context)!;
    final messenger = ScaffoldMessenger.of(context);
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        content: Text(l10n.chatDeleteMessageConfirm),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(ctx).pop(false),
            child: Text(l10n.commonCancel),
          ),
          FilledButton(
            style: FilledButton.styleFrom(
              backgroundColor: Theme.of(ctx).colorScheme.error,
            ),
            onPressed: () => Navigator.of(ctx).pop(true),
            child: Text(l10n.commonDelete),
          ),
        ],
      ),
    );
    if (confirmed != true) {
      return;
    }
    try {
      await ref.read(chatControllerProvider.notifier).deleteMessage(message.id);
    } on Object catch (e) {
      messenger.showSnackBar(
        SnackBar(content: Text(chatErrorMessage(l10n, e))),
      );
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final theme = Theme.of(context);
    final asyncState = ref.watch(chatControllerProvider);

    return Scaffold(
      key: _scaffoldKey,
      drawer: ChatDrawer(
        onLogout: () {
          Navigator.of(context).pop();
          _logout();
        },
        onDeleteAccount: () {
          Navigator.of(context).pop();
          _confirmDeleteAccount();
        },
        onOpenPrivacy: () => _openUrl(AppConfig.privacyPolicyUrl),
        onOpenTerms: () => _openUrl(AppConfig.termsOfServiceUrl),
      ),
      appBar: AppBar(
        leading: IconButton(
          icon: const Icon(Icons.menu_rounded),
          tooltip: l10n.chatMenu,
          onPressed: () => _scaffoldKey.currentState?.openDrawer(),
        ),
        title: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            const BrandLogo(size: 24),
            const SizedBox(width: 8),
            Text(l10n.appTitle, style: theme.appBarTheme.titleTextStyle),
            const SizedBox(width: 2),
            Icon(
              Icons.keyboard_arrow_down_rounded,
              size: 20,
              color: theme.palette.textSubtle,
            ),
          ],
        ),
        actions: [
          IconButton(
            icon: const Icon(Icons.edit_outlined),
            tooltip: l10n.chatNewConversation,
            onPressed: () {
              _inputFocus.requestFocus();
              _scrollToLatest();
            },
          ),
        ],
      ),
      body: SafeArea(
        top: false,
        child: Column(
          children: [
            Expanded(
              child: asyncState.when(
                loading: () => const Center(child: CircularProgressIndicator()),
                error: (_, _) => Center(
                  child: Padding(
                    padding: const EdgeInsets.symmetric(horizontal: 32),
                    child: Text(
                      l10n.chatErrorGeneric,
                      textAlign: TextAlign.center,
                    ),
                  ),
                ),
                data: (state) => _messageList(l10n, state),
              ),
            ),
            const AttachmentStrip(),
            _inputBar(l10n, asyncState.valueOrNull?.sending ?? false),
          ],
        ),
      ),
    );
  }

  Widget _messageList(AppLocalizations l10n, ChatState state) {
    if (state.messages.isEmpty) {
      return ChatEmptyState(onSuggestion: _onSuggestion);
    }
    // A user message is "analysed" once a completed assistant turn follows it
    // (stamps the photo badge); an assistant turn "reviewed a photo" when the
    // preceding user turn carried an image (shows the author subtitle).
    final analysedUserIds = <String>{};
    final reviewedPhotoIds = <String>{};
    bool hasImage(ChatMessage m) =>
        m.content.any((b) => b.type == 'image' && b.storageKey.isNotEmpty);
    for (var i = 0; i < state.messages.length; i++) {
      final m = state.messages[i];
      if (m.role == MessageRole.user && i + 1 < state.messages.length) {
        final next = state.messages[i + 1];
        if (next.role == MessageRole.assistant &&
            next.status == MessageStatus.complete) {
          analysedUserIds.add(m.id);
        }
      }
      if (m.role == MessageRole.assistant && i > 0) {
        final prev = state.messages[i - 1];
        if (prev.role == MessageRole.user && hasImage(prev)) {
          reviewedPhotoIds.add(m.id);
        }
      }
    }
    final reversed = state.messages.reversed.toList();
    return ListView.builder(
      controller: _scroll,
      reverse: true,
      padding: const EdgeInsets.only(top: 8, bottom: 8),
      itemCount: reversed.length + (state.loadingOlder ? 1 : 0),
      itemBuilder: (context, index) {
        if (index == reversed.length) {
          return const Padding(
            padding: EdgeInsets.all(12),
            child: Center(
              child: SizedBox(
                height: 20,
                width: 20,
                child: CircularProgressIndicator(strokeWidth: 2),
              ),
            ),
          );
        }
        final message = reversed[index];
        final isAssistant = message.role == MessageRole.assistant;
        final isComplete = message.status == MessageStatus.complete;
        // The newest assistant turn (reversed index 0) may be regenerated.
        final isLatest = index == 0;
        return GestureDetector(
          onLongPress: () => _confirmDeleteMessage(message),
          child: MessageBubble(
            message: message,
            photoAnalysed: analysedUserIds.contains(message.id),
            reviewedPhoto: reviewedPhotoIds.contains(message.id),
            showFooter: isAssistant && isComplete && !message.streaming,
            feedbackValue: state.feedback[message.id],
            onFeedback: (value) => ref
                .read(chatControllerProvider.notifier)
                .setFeedback(message.id, value),
            onRegenerate: (isLatest && !state.sending)
                ? () => ref.read(chatControllerProvider.notifier).regenerate()
                : null,
            onRetry: message.status == MessageStatus.failed
                ? () => ref.read(chatControllerProvider.notifier).retry()
                : null,
            onFertilizerTap: (product) => ref
                .read(chatControllerProvider.notifier)
                .recordFertilizerTap(product.slug),
          ),
        );
      },
    );
  }

  Widget _inputBar(AppLocalizations l10n, bool sending) {
    final theme = Theme.of(context);
    final p = theme.palette;
    final composer = ref.watch(messageComposerProvider);
    final uploading = composer.uploading;
    final voicePhase = ref.watch(voiceRecorderProvider).phase;
    final attached = composer.hasAttachments;

    // A recorded voice note awaiting confirmation (or being sent) takes over the
    // whole input row.
    if (voicePhase == VoicePhase.preview ||
        voicePhase == VoicePhase.uploading) {
      return _composerShell(p, const VoiceRecorderBar());
    }

    final recording = voicePhase == VoicePhase.recording;
    if (recording) {
      // The level/waveform display takes the row, but the mic button stays
      // mounted — the active long-press gesture lives on it.
      return _composerShell(
        p,
        Row(
          crossAxisAlignment: CrossAxisAlignment.center,
          children: [
            const Expanded(child: VoiceRecorderBar()),
            const SizedBox(width: 2),
            _micButton(l10n, true),
          ],
        ),
      );
    }

    return _composerShell(
      p,
      Row(
        crossAxisAlignment: CrossAxisAlignment.end,
        children: [
          _circleIcon(
            icon: Icons.add_rounded,
            color: p.text,
            onTap: (sending || uploading || !composer.canAddMore)
                ? null
                : _showAttachSheet,
            tooltip: l10n.chatAttach,
          ),
          Expanded(
            child: TextField(
              controller: _input,
              focusNode: _inputFocus,
              minLines: 1,
              maxLines: 5,
              textInputAction: TextInputAction.send,
              onSubmitted: (_) => _send(),
              style: theme.textTheme.bodyLarge,
              decoration: InputDecoration(
                isCollapsed: true,
                filled: false,
                border: InputBorder.none,
                enabledBorder: InputBorder.none,
                focusedBorder: InputBorder.none,
                contentPadding: const EdgeInsets.symmetric(
                  horizontal: 6,
                  vertical: 10,
                ),
                hintText: attached
                    ? l10n.chatInputPlaceholderPhoto
                    : l10n.chatInputPlaceholder,
              ),
            ),
          ),
          const SizedBox(width: 2),
          _trailingButton(l10n, sending, uploading, recording),
        ],
      ),
    );
  }

  /// The rounded pill that wraps the composer row (or the voice recorder).
  Widget _composerShell(AppPalette p, Widget child) {
    return Container(
      padding: const EdgeInsets.fromLTRB(12, 8, 12, 12),
      decoration: BoxDecoration(
        color: p.surface,
        border: Border(top: BorderSide(color: p.border)),
      ),
      child: Container(
        padding: const EdgeInsets.all(6),
        decoration: BoxDecoration(
          color: p.inputBg,
          border: Border.all(color: p.inputBorder),
          borderRadius: BorderRadius.circular(AppRadius.composer),
        ),
        child: child,
      ),
    );
  }

  Widget _circleIcon({
    required IconData icon,
    required Color color,
    required VoidCallback? onTap,
    String? tooltip,
  }) {
    return IconButton(
      onPressed: onTap,
      icon: Icon(icon, color: onTap == null ? color.withValues(alpha: 0.4) : color),
      tooltip: tooltip,
      visualDensity: VisualDensity.compact,
    );
  }

  Widget _trailingButton(
    AppLocalizations l10n,
    bool sending,
    bool uploading,
    bool recording,
  ) {
    final p = Theme.of(context).palette;
    if (sending) {
      return _filledCircle(
        bg: p.soft,
        child: Icon(Icons.stop_rounded, color: p.text, size: 22),
        onTap: () => ref.read(chatControllerProvider.notifier).cancel(),
        tooltip: l10n.chatStop,
      );
    }
    if (uploading) {
      return _filledCircle(
        bg: p.accent,
        child: SizedBox(
          height: 18,
          width: 18,
          child: CircularProgressIndicator(strokeWidth: 2, color: p.onAccent),
        ),
        onTap: null,
        tooltip: l10n.chatSend,
      );
    }
    return ValueListenableBuilder<TextEditingValue>(
      valueListenable: _input,
      builder: (context, value, _) {
        if (!recording && value.text.trim().isNotEmpty) {
          return _filledCircle(
            bg: p.accent,
            child: Icon(Icons.arrow_upward_rounded, color: p.onAccent, size: 22),
            onTap: _send,
            tooltip: l10n.chatSend,
          );
        }
        return _micButton(l10n, recording);
      },
    );
  }

  Widget _filledCircle({
    required Color bg,
    required Widget child,
    required VoidCallback? onTap,
    String? tooltip,
  }) {
    final btn = Material(
      color: bg,
      shape: const CircleBorder(),
      child: InkWell(
        onTap: onTap,
        customBorder: const CircleBorder(),
        child: SizedBox(width: 38, height: 38, child: Center(child: child)),
      ),
    );
    return tooltip == null ? btn : Tooltip(message: tooltip, child: btn);
  }

  Widget _micButton(AppLocalizations l10n, bool recording) {
    final p = Theme.of(context).palette;
    return GestureDetector(
      onTap: () => ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text(l10n.voiceHoldToRecord),
          duration: const Duration(seconds: 1),
        ),
      ),
      onLongPressStart: (_) => _startVoice(),
      onLongPressMoveUpdate: (d) => ref
          .read(voiceRecorderProvider.notifier)
          .updateDrag(d.offsetFromOrigin.dx),
      onLongPressEnd: (_) => _endVoice(),
      child: Container(
        width: 38,
        height: 38,
        decoration: BoxDecoration(
          color: recording ? p.danger : p.soft,
          shape: BoxShape.circle,
        ),
        child: Icon(
          Icons.mic_none_rounded,
          color: recording ? Colors.white : p.text,
          size: 22,
        ),
      ),
    );
  }

  Future<void> _startVoice() async {
    final result = await ref
        .read(voiceRecorderProvider.notifier)
        .startRecording();
    if (!mounted) {
      return;
    }
    switch (result) {
      case VoiceStartResult.started:
      case VoiceStartResult.failed:
        break;
      case VoiceStartResult.permissionDenied:
      case VoiceStartResult.permissionPermanentlyDenied:
        await _showMicPermissionDialog();
    }
  }

  Future<void> _endVoice() async {
    final notifier = ref.read(voiceRecorderProvider.notifier);
    if (ref.read(voiceRecorderProvider).cancelArmed) {
      await notifier.cancelRecording();
    } else {
      await notifier.stopToPreview();
    }
  }

  Future<void> _showMicPermissionDialog() async {
    final l10n = AppLocalizations.of(context)!;
    final open = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text(l10n.voicePermissionTitle),
        content: Text(l10n.voicePermissionBody),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(ctx).pop(false),
            child: Text(l10n.commonCancel),
          ),
          FilledButton(
            onPressed: () => Navigator.of(ctx).pop(true),
            child: Text(l10n.chatOpenSettings),
          ),
        ],
      ),
    );
    if (open == true) {
      await ref.read(permissionPortProvider).openSettings();
    }
  }
}
