import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'

export default function App() {
  const { t, i18n } = useTranslation()
  return (
    <div className="min-h-svh bg-background text-foreground">
      <header className="mx-auto flex max-w-5xl items-center justify-between gap-4 px-6 py-6">
        <span className="text-xl font-semibold tracking-tight">PicketX</span>
        <div role="group" aria-label={t('language')} className="flex gap-2">
          <Button variant="ghost" aria-pressed={i18n.language === 'en'} onClick={() => void i18n.changeLanguage('en')}>English</Button>
          <Button variant="ghost" lang="zh-CN" aria-pressed={i18n.language === 'zh-CN'} onClick={() => void i18n.changeLanguage('zh-CN')}>简体中文</Button>
        </div>
      </header>
      <main className="mx-auto max-w-5xl space-y-10 px-6 py-12 sm:py-20">
        <div className="max-w-2xl space-y-5">
          <p className="text-sm font-medium text-muted-foreground">{t('tagline')}</p>
          <h1 className="text-4xl font-semibold tracking-tight sm:text-5xl">{t('title')}</h1>
          <p className="text-lg leading-relaxed text-muted-foreground">{t('description')}</p>
        </div>
        <Card>
          <CardHeader>
            <CardTitle>{t('status')}</CardTitle>
            <CardDescription>{t('statusDescription')}</CardDescription>
          </CardHeader>
        </Card>
        <div className="grid gap-5 sm:grid-cols-2">
          {(['portal', 'admin'] as const).map((area) => (
            <Card key={area}>
              <CardHeader>
                <CardTitle>{t(area)}</CardTitle>
                <CardDescription>{t(`${area}Description`)}</CardDescription>
              </CardHeader>
              <CardContent><span className="text-sm text-muted-foreground">{t('planned')}</span></CardContent>
            </Card>
          ))}
        </div>
      </main>
      <footer className="mx-auto max-w-5xl px-6 py-8 text-sm text-muted-foreground">{t('footer')}</footer>
    </div>
  )
}
