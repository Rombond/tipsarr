package com.brebond.tipsarr

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.activity.viewModels
import com.brebond.tipsarr.design.TipsarrTheme

class MainActivity : ComponentActivity() {
    private val viewModel: SessionViewModel by viewModels()

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()
        setContent {
            TipsarrTheme {
                Root(viewModel.session)
            }
        }
    }
}
