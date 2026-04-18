using System;
using System.Linq;
using System.Threading;
using Windows.Data.Xml.Dom;
using Windows.UI.Notifications;

class Program
{
    static int Main(string[] args)
    {
        string title = "";
        string message = "";
        string appId = "{1AC14E77-02E7-4E5D-B744-2EB1AE5198B7}\\WindowsPowerShell\\v1.0\\powershell.exe";
        string[] actionKeys = Array.Empty<string>();
        string[] actionLabels = Array.Empty<string>();
        int timeout = 30;

        for (int i = 0; i < args.Length; i++)
        {
            switch (args[i])
            {
                case "--title" when i + 1 < args.Length:
                    title = args[++i];
                    break;
                case "--message" when i + 1 < args.Length:
                    message = args[++i];
                    break;
                case "--app-id" when i + 1 < args.Length:
                    appId = args[++i];
                    break;
                case "--action-keys" when i + 1 < args.Length:
                    actionKeys = args[++i].Split(',');
                    break;
                case "--action-labels" when i + 1 < args.Length:
                    actionLabels = args[++i].Split(',');
                    break;
                case "--timeout" when i + 1 < args.Length:
                    int.TryParse(args[++i], out timeout);
                    break;
            }
        }

        // Build toast XML
        string actionsXml = "";
        if (actionKeys.Length > 0 && actionKeys.Length == actionLabels.Length)
        {
            actionsXml = "<actions>";
            for (int i = 0; i < actionKeys.Length; i++)
            {
                actionsXml += $"<action content=\"{Escape(actionLabels[i])}\" arguments=\"{Escape(actionKeys[i])}\" activationType=\"foreground\"/>";
            }
            actionsXml += "</actions>";
        }

        string toastXml = $@"<toast>
  <visual>
    <binding template=""ToastGeneric"">
      <text>{Escape(title)}</text>
      <text>{Escape(message)}</text>
    </binding>
  </visual>
  {actionsXml}
</toast>";

        var xml = new XmlDocument();
        xml.LoadXml(toastXml);

        var toast = new ToastNotification(xml);

        string result = "";
        var done = new ManualResetEventSlim(false);

        toast.Activated += (sender, e) =>
        {
            if (e is ToastActivatedEventArgs activated)
            {
                result = activated.Arguments;
            }
            done.Set();
        };

        toast.Dismissed += (sender, e) =>
        {
            result = "__dismissed__";
            done.Set();
        };

        toast.Failed += (sender, e) =>
        {
            result = "__failed__";
            done.Set();
        };

        var notifier = ToastNotificationManager.CreateToastNotifier(appId);
        notifier.Show(toast);

        if (actionKeys.Length > 0)
        {
            done.Wait(TimeSpan.FromSeconds(timeout));
            Console.Write(result);
        }

        return 0;
    }

    static string Escape(string s)
    {
        return s.Replace("&", "&amp;")
                .Replace("<", "&lt;")
                .Replace(">", "&gt;")
                .Replace("\"", "&quot;")
                .Replace("'", "&apos;");
    }
}
