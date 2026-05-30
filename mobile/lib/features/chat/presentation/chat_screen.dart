import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../l10n/generated/app_localizations.dart';
import '../../auth/application/auth_controller.dart';
import '../../auth/presentation/auth_error_message.dart';
import '../application/chat_controller.dart';
import '../application/message_composer.dart';
import '../data/media_ports.dart';
import '../domain/chat_models.dart';
import 'chat_error_message.dart';
import 'message_bubble.dart';
import 'widgets/attachment_strip.dart';

enum _ChatMenuAction { logout, deleteAccount }

enum _AttachSource { camera, gallery }

class ChatScreen extends ConsumerStatefulWidget {
  const ChatScreen({super.key});

  @override
  ConsumerState<ChatScreen> createState() => _ChatScreenState();
}

class _ChatScreenState extends ConsumerState<ChatScreen> {
  final _input = TextEditingController();
  final _scroll = ScrollController();

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
    super.dispose();
  }

  void _onScroll() {
    // reverse:true → older messages sit at the top, i.e. max scroll extent.
    if (_scroll.position.pixels >= _scroll.position.maxScrollExtent - 200) {
      ref.read(chatControllerProvider.notifier).loadOlder();
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

  Future<void> _showAttachSheet() async {
    final l10n = AppLocalizations.of(context)!;
    final source = await showModalBottomSheet<_AttachSource>(
      context: context,
      builder: (ctx) => SafeArea(
        child: Wrap(
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
        messenger.showSnackBar(
          SnackBar(content: Text(l10n.chatErrorGeneric)),
        );
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
    final asyncState = ref.watch(chatControllerProvider);

    return Scaffold(
      appBar: AppBar(
        title: Text(l10n.chatTitle),
        actions: [
          PopupMenuButton<_ChatMenuAction>(
            onSelected: (action) {
              switch (action) {
                case _ChatMenuAction.logout:
                  _logout();
                case _ChatMenuAction.deleteAccount:
                  _confirmDeleteAccount();
              }
            },
            itemBuilder: (context) => [
              PopupMenuItem(
                value: _ChatMenuAction.logout,
                child: Text(l10n.chatLogout),
              ),
              PopupMenuItem(
                value: _ChatMenuAction.deleteAccount,
                child: Text(l10n.chatDeleteAccount),
              ),
            ],
          ),
        ],
      ),
      body: SafeArea(
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
      return Center(
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 32),
          child: Text(
            l10n.chatEmptyHint,
            textAlign: TextAlign.center,
            style: Theme.of(context).textTheme.bodyLarge,
          ),
        ),
      );
    }
    final reversed = state.messages.reversed.toList();
    return ListView.builder(
      controller: _scroll,
      reverse: true,
      padding: const EdgeInsets.symmetric(vertical: 8),
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
        return GestureDetector(
          onLongPress: () => _confirmDeleteMessage(message),
          child: MessageBubble(
            message: message,
            onRetry: message.status == MessageStatus.failed
                ? () => ref.read(chatControllerProvider.notifier).retry()
                : null,
          ),
        );
      },
    );
  }

  Widget _inputBar(AppLocalizations l10n, bool sending) {
    final composer = ref.watch(messageComposerProvider);
    final uploading = composer.uploading;
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
      child: Row(
        children: [
          IconButton(
            onPressed: (sending || uploading || !composer.canAddMore)
                ? null
                : _showAttachSheet,
            icon: const Icon(Icons.attach_file_rounded),
            tooltip: l10n.chatAttach,
          ),
          Expanded(
            child: TextField(
              controller: _input,
              minLines: 1,
              maxLines: 5,
              textInputAction: TextInputAction.send,
              onSubmitted: (_) => _send(),
              decoration: InputDecoration(
                hintText: l10n.chatInputPlaceholder,
                border: const OutlineInputBorder(),
              ),
            ),
          ),
          const SizedBox(width: 8),
          if (sending)
            IconButton.filledTonal(
              onPressed: () =>
                  ref.read(chatControllerProvider.notifier).cancel(),
              icon: const Icon(Icons.stop_rounded),
              tooltip: l10n.chatStop,
            )
          else if (uploading)
            IconButton.filled(
              onPressed: null,
              icon: const SizedBox(
                height: 18,
                width: 18,
                child: CircularProgressIndicator(strokeWidth: 2),
              ),
              tooltip: l10n.chatSend,
            )
          else
            IconButton.filled(
              onPressed: _send,
              icon: const Icon(Icons.send_rounded),
              tooltip: l10n.chatSend,
            ),
        ],
      ),
    );
  }
}
